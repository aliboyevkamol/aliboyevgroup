package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kamolportfolio/backend/internal/auth"
	"kamolportfolio/backend/pkg/response"
)

// ---------- contact ----------

func (s *Server) contact(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Name, Email, Phone, Subject, Message string }
	if err := decode(r, &in); err != nil {
		return err
	}
	v := newV()
	in.Name = v.str("name", in.Name, 1, 100)
	in.Email = v.email("email", in.Email)
	in.Subject = v.str("subject", in.Subject, 0, 200)
	in.Message = v.str("message", in.Message, 3, 5000)
	if !phoneRe.MatchString(strings.TrimSpace(in.Phone)) {
		v.f["phone"] = "invalid phone number"
	}
	if err := v.err(); err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO contact_messages (name,email,phone,subject,message) VALUES ($1,$2,$3,$4,$5)`, in.Name, in.Email, strings.TrimSpace(in.Phone), in.Subject, in.Message); err != nil {
		return err
	}
	response.OK(w, 201, map[string]string{"message": "Thanks! Your message was received."})
	return nil
}

func (s *Server) listContact(w http.ResponseWriter, r *http.Request) error {
	f := &filter{}
	if st := strings.ToUpper(r.URL.Query().Get("status")); st != "" {
		f.add("m.status = ?", st)
	}
	return s.paged(w, r, `json_build_object('id',m.id,'name',m.name,'email',m.email,'phone',m.phone,'subject',m.subject,'message',m.message,'status',m.status,'created_at',m.created_at)`, "FROM contact_messages m"+f.clause(), "m.created_at DESC, m.id DESC", f.args, 50)
}

func (s *Server) setContact(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct{ Status string }
	if err := decode(r, &in); err != nil {
		return err
	}
	st := strings.ToUpper(in.Status)
	if st != "NEW" && st != "READ" && st != "REPLIED" && st != "ARCHIVED" {
		return response.Validation(map[string]string{"status": "must be NEW, READ, REPLIED or ARCHIVED"})
	}
	res, err := s.DB.ExecContext(r.Context(), `UPDATE contact_messages SET status=$1 WHERE id=$2`, st, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return response.E(404, "MESSAGE_NOT_FOUND", "Message not found.")
	}
	response.OK(w, 200, map[string]string{"status": st})
	return nil
}

// ---------- portfolio content ----------

func (s *Server) portfolio(w http.ResponseWriter, r *http.Request) error {
	set := func(k string) string { return `(SELECT value FROM site_settings WHERE key='` + k + `')` }
	return s.one(w, r, "NOT_FOUND", `SELECT json_build_object('hero_title',`+set("hero_title")+`,'hero_description',`+set("hero_description")+`,'about',`+set("about")+`,'contact',`+set("contact")+`,'social_links',`+set("social_links")+`,
'skills',(SELECT coalesce(json_agg(json_build_object('id',id,'category',category,'name',name,'level',level) ORDER BY position,id),'[]'::json) FROM skills),
'services',(SELECT coalesce(json_agg(json_build_object('id',id,'title',title,'description',description) ORDER BY position,id),'[]'::json) FROM services),
'experiences',(SELECT coalesce(json_agg(json_build_object('id',id,'title',title,'description',description,'period',period) ORDER BY position,id),'[]'::json) FROM experiences))`)
}

func (s *Server) savePortfolio(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		HeroTitle       *string           `json:"hero_title"`
		HeroDescription *string           `json:"hero_description"`
		About           json.RawMessage   `json:"about"`
		Contact         json.RawMessage   `json:"contact"`
		SocialLinks     map[string]string `json:"social_links"`
		Skills          *[]struct {
			Category, Name string
			Level          int
		} `json:"skills"`
		Services    *[]struct{ Title, Description string }         `json:"services"`
		Experiences *[]struct{ Title, Description, Period string } `json:"experiences"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	v := newV()
	if in.HeroTitle != nil {
		*in.HeroTitle = v.str("hero_title", *in.HeroTitle, 1, 200)
	}
	if in.HeroDescription != nil {
		*in.HeroDescription = v.str("hero_description", *in.HeroDescription, 0, 1000)
	}
	for k, u := range in.SocialLinks {
		if k == "email" {
			if u != "" && !strings.HasPrefix(u, "mailto:") {
				v.f["social_links.email"] = "must start with mailto:"
			}
			continue
		}
		in.SocialLinks[k] = v.url("social_links."+k, u, false)
	}
	if in.Skills != nil {
		if len(*in.Skills) > 200 {
			v.f["skills"] = "too many skills"
		}
		for _, k := range *in.Skills {
			if k.Name == "" || len(k.Name) > 60 || k.Category == "" || len(k.Category) > 30 || k.Level < 0 || k.Level > 100 {
				v.f["skills"] = "each skill needs a name, a category and a level between 0 and 100"
			}
		}
	}
	if err := v.err(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	put := func(k string, val any) error {
		b, _ := json.Marshal(val)
		_, err := tx.ExecContext(r.Context(), `INSERT INTO site_settings (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, updated_at=now()`, k, string(b))
		return err
	}
	if in.HeroTitle != nil {
		if err := put("hero_title", *in.HeroTitle); err != nil {
			return err
		}
	}
	if in.HeroDescription != nil {
		if err := put("hero_description", *in.HeroDescription); err != nil {
			return err
		}
	}
	if in.SocialLinks != nil {
		if err := put("social_links", in.SocialLinks); err != nil {
			return err
		}
	}
	for k, raw := range map[string]json.RawMessage{"about": in.About, "contact": in.Contact} {
		if len(raw) > 0 && string(raw) != "null" {
			if len(raw) > 20000 || !json.Valid(raw) {
				return response.Validation(map[string]string{k: "invalid value"})
			}
			if _, err := tx.ExecContext(r.Context(), `INSERT INTO site_settings (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, updated_at=now()`, k, string(raw)); err != nil {
				return err
			}
		}
	}
	if in.Skills != nil {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM skills`); err != nil {
			return err
		}
		for i, k := range *in.Skills {
			if _, err := tx.ExecContext(r.Context(), `INSERT INTO skills (category,name,level,position) VALUES ($1,$2,$3,$4)`, strings.ToLower(k.Category), k.Name, k.Level, i); err != nil {
				return err
			}
		}
	}
	if in.Services != nil {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM services`); err != nil {
			return err
		}
		for i, k := range *in.Services {
			if _, err := tx.ExecContext(r.Context(), `INSERT INTO services (title,description,position) VALUES ($1,$2,$3)`, k.Title, k.Description, i); err != nil {
				return err
			}
		}
	}
	if in.Experiences != nil {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM experiences`); err != nil {
			return err
		}
		for i, k := range *in.Experiences {
			if _, err := tx.ExecContext(r.Context(), `INSERT INTO experiences (title,description,period,position) VALUES ($1,$2,$3,$4)`, k.Title, k.Description, k.Period, i); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.audit(r, "portfolio updated", "portfolio", "site", nil)
	return s.portfolio(w, r)
}

// ---------- media ----------

var mediaTypes = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}
var privateTypes = map[string]string{"application/zip": ".zip", "application/pdf": ".pdf"}

func (s *Server) uploadMedia(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return response.E(413, "UPLOAD_TOO_LARGE", "The file is too large or the form is invalid.")
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return response.Validation(map[string]string{"file": "is required"})
	}
	defer file.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	ct := http.DetectContentType(head[:n])
	private := r.FormValue("private") == "true"
	ext, ok := mediaTypes[ct]
	if !ok && private {
		ext, ok = privateTypes[ct]
	}
	if !ok {
		return response.E(415, "UNSUPPORTED_TYPE", "Allowed: PNG, JPEG, GIF, WebP images (and ZIP/PDF as private files).")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	prefix := "public/"
	if private || privateTypes[ct] != "" {
		prefix, private = "private/", true
	}
	key := prefix + auth.RandomToken(16) + ext
	if err := s.Store.Upload(r.Context(), key, file); err != nil {
		return err
	}
	url := ""
	if !private {
		url, _ = s.Store.GetURL(key, 0)
	}
	name := filepath.Base(hdr.Filename)
	if len(name) > 200 {
		name = name[:200]
	}
	b, err := s.raw(r.Context(), `INSERT INTO media (uploader_id,storage_key,url,filename,mime_type,size_bytes,is_private) VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING json_build_object('id',id,'storage_key',storage_key,'url',url,'filename',filename,'mime_type',mime_type,'size_bytes',size_bytes,'is_private',is_private,'created_at',created_at)`,
		uid(r), key, url, name, ct, hdr.Size, private)
	if err != nil {
		return err
	}
	s.audit(r, "media uploaded", "media", key, nil)
	response.OK(w, 201, b)
	return nil
}

func (s *Server) listMedia(w http.ResponseWriter, r *http.Request) error {
	return s.paged(w, r, `json_build_object('id',m.id,'storage_key',m.storage_key,'url',m.url,'filename',m.filename,'mime_type',m.mime_type,'size_bytes',m.size_bytes,'is_private',m.is_private,'created_at',m.created_at)`, "FROM media m", "m.created_at DESC, m.id DESC", nil, 30)
}

func (s *Server) deleteMedia(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var key string
	if err := s.DB.QueryRowContext(r.Context(), `DELETE FROM media WHERE id=$1 RETURNING storage_key`, id).Scan(&key); err != nil {
		return response.E(404, "MEDIA_NOT_FOUND", "Not found.")
	}
	_ = s.Store.Delete(r.Context(), key)
	s.audit(r, "media deleted", "media", id, nil)
	response.NoContent(w)
	return nil
}

// serveFile serves local-storage files: "public/" keys openly, everything else only with a valid short-lived signature.
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, err := s.Store.Open(r.PathValue("key"), q.Get("exp"), q.Get("sig"))
	if errors.Is(err, os.ErrPermission) {
		response.Fail(w, response.E(403, "LINK_EXPIRED", "This download link is invalid or has expired."))
		return
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, _ := f.Stat()
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !strings.HasPrefix(r.PathValue("key"), "public/") {
		w.Header().Set("Content-Disposition", "attachment")
		w.Header().Set("Cache-Control", "private, no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	http.ServeContent(w, r, filepath.Base(r.PathValue("key")), st.ModTime(), f)
}

var _ = time.Now
