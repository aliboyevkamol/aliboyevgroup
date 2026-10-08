package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"kamolportfolio/backend/pkg/response"
	"kamolportfolio/backend/pkg/slug"
)

var categories = map[string]bool{"websites": true, "saas": true, "bots": true, "ai": true, "templates": true, "automation": true}

// ---------- shared SQL helpers ----------

func insertRow(ctx context.Context, tx *sql.Tx, table string, cols map[string]any) (int64, error) {
	keys := make([]string, 0, len(cols))
	for k := range cols {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ph, args := make([]string, len(keys)), make([]any, len(keys))
	for i, k := range keys {
		ph[i], args[i] = "$"+strconv.Itoa(i+1), cols[k]
	}
	var id int64
	err := tx.QueryRowContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", table, strings.Join(keys, ","), strings.Join(ph, ",")), args...).Scan(&id)
	return id, err
}

func updateRow(ctx context.Context, tx *sql.Tx, table string, id int64, cols map[string]any) (bool, error) {
	keys := make([]string, 0, len(cols))
	for k := range cols {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sets, args := []string{"updated_at=now()"}, []any{}
	for i, k := range keys {
		sets = append(sets, fmt.Sprintf("%s=$%d", k, i+1))
		args = append(args, cols[k])
	}
	args = append(args, id)
	res, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET %s WHERE id=$%d", table, strings.Join(sets, ","), len(args)), args...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func setList(ctx context.Context, tx *sql.Tx, table, fk, col string, id int64, vals []string) error {
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s=$1", table, fk), id); err != nil {
		return err
	}
	seen := map[string]bool{}
	pos := 0
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" || len(v) > 500 || seen[strings.ToLower(v)] {
			continue
		}
		seen[strings.ToLower(v)] = true
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (%s,%s,position) VALUES ($1,$2,$3)", table, fk, col), id, v, pos); err != nil {
			return err
		}
		pos++
	}
	return nil
}

func uniqueSlug(ctx context.Context, tx *sql.Tx, table, base string) (string, error) {
	cand := base
	for i := 2; i < 1000; i++ {
		var x int
		err := tx.QueryRowContext(ctx, fmt.Sprintf("SELECT 1 FROM %s WHERE slug=$1", table), cand).Scan(&x)
		if err == sql.ErrNoRows {
			return cand, nil
		}
		if err != nil {
			return "", err
		}
		cand = fmt.Sprintf("%s-%d", base, i)
	}
	return "", fmt.Errorf("no free slug")
}

// refWhere builds "alias.id=$n" for numeric refs and "alias.slug=$n" otherwise.
func refWhere(ref, alias string, n int) (string, any) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return fmt.Sprintf("%s.id=$%d", alias, n), id
	}
	return fmt.Sprintf("%s.slug=$%d", alias, n), ref
}

type filter struct {
	where []string
	args  []any
}

func (f *filter) add(cond string, v any) {
	f.args = append(f.args, v)
	f.where = append(f.where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(f.args))))
}
func (f *filter) clause() string {
	if len(f.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(f.where, " AND ")
}

func sortBy(r *http.Request, m map[string]string, def string) string {
	if v, ok := m[r.URL.Query().Get("sort")]; ok {
		return v
	}
	return def
}

func (s *Server) adminDelete(table, entity string) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		res, err := s.DB.ExecContext(r.Context(), "DELETE FROM "+table+" WHERE id=$1", id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return response.E(404, strings.ToUpper(entity)+"_NOT_FOUND", "Not found.")
		}
		s.audit(r, entity+" deleted", entity, id, nil)
		response.NoContent(w)
		return nil
	}
}

const listAgg = `(SELECT coalesce(json_agg(%[2]s ORDER BY t.position),'[]'::json) FROM %[1]s t WHERE t.%[3]s=p.id)`

func listSQL(table, col, fk string) string { return fmt.Sprintf(listAgg, table, "t."+col, fk) }

// ---------- products ----------

func productExpr(admin bool) string {
	extra := `'has_download',p.download_url<>''`
	if admin {
		extra += `,'download_url',p.download_url`
	}
	return `json_build_object('id',p.id,'title',p.title,'slug',p.slug,'category',p.category,'description',p.description,'content',p.content,
'price',p.price,'old_price',p.old_price,'currency',p.currency,'cover_image',p.cover_image,'demo_url',p.demo_url,'demo_type',p.demo_type,
'live_url',p.live_url,'mock_variant',p.mock_variant,'featured',p.featured,'published',p.published,` + extra + `,
'technologies',` + listSQL("product_technologies", "name", "product_id") + `,'features',` + listSQL("product_features", "text", "product_id") + `,
'created_at',p.created_at,'updated_at',p.updated_at)`
}

func (s *Server) listProducts(admin bool) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.URL.Query()
		f := &filter{}
		if !admin {
			f.where = append(f.where, "p.published")
		} else if v := q.Get("published"); v != "" {
			f.add("p.published = ?", v == "true")
		}
		if c := q.Get("category"); c != "" {
			if !categories[c] {
				return response.Validation(map[string]string{"category": "unknown category"})
			}
			f.add("p.category = ?", c)
		}
		if q.Get("featured") == "true" {
			f.where = append(f.where, "p.featured")
		}
		if sr := strings.TrimSpace(q.Get("search")); sr != "" {
			f.add(`(p.title ILIKE ? OR p.description ILIKE ? OR EXISTS(SELECT 1 FROM product_technologies t WHERE t.product_id=p.id AND t.name ILIKE ?))`, "%"+escapeLike(sr)+"%")
		}
		order := sortBy(r, map[string]string{"newest": "p.created_at DESC, p.id DESC", "oldest": "p.created_at, p.id", "price_asc": "p.price, p.id", "price_desc": "p.price DESC, p.id", "title": "p.title, p.id"}, "p.created_at DESC, p.id DESC")
		return s.paged(w, r, productExpr(admin), "FROM products p"+f.clause(), order, f.args, 12)
	}
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) error {
	cond, arg := refWhere(r.PathValue("ref"), "p", 1)
	return s.one(w, r, "PRODUCT_NOT_FOUND", "SELECT "+productExpr(false)+" FROM products p WHERE p.published AND "+cond, arg)
}

func (s *Server) adminGetProduct(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return s.one(w, r, "PRODUCT_NOT_FOUND", "SELECT "+productExpr(true)+" FROM products p WHERE p.id=$1", id)
}

type productIn struct {
	Title        *string   `json:"title"`
	Slug         *string   `json:"slug"`
	Category     *string   `json:"category"`
	Description  *string   `json:"description"`
	Content      *string   `json:"content"`
	Price        *float64  `json:"price"`
	OldPrice     *float64  `json:"old_price"`
	Currency     *string   `json:"currency"`
	CoverImage   *string   `json:"cover_image"`
	DemoURL      *string   `json:"demo_url"`
	DemoType     *string   `json:"demo_type"`
	LiveURL      *string   `json:"live_url"`
	DownloadURL  *string   `json:"download_url"`
	MockVariant  *int      `json:"mock_variant"`
	Featured     *bool     `json:"featured"`
	Published    *bool     `json:"published"`
	Technologies *[]string `json:"technologies"`
	Features     *[]string `json:"features"`
}

func (s *Server) saveProduct(patch bool) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		var id int64
		if patch {
			var err error
			if id, err = pathID(r, "id"); err != nil {
				return err
			}
		}
		var in productIn
		if err := decode(r, &in); err != nil {
			return err
		}
		v := newV()
		cols := map[string]any{}
		if !patch {
			if in.Title == nil {
				v.f["title"] = "is required"
			}
			if in.Category == nil {
				v.f["category"] = "is required"
			}
			if in.Price == nil {
				v.f["price"] = "is required"
			}
		}
		str := func(k string, p *string, min, max int) {
			if p != nil {
				cols[k] = v.str(k, *p, min, max)
			}
		}
		str("title", in.Title, 1, 200)
		str("description", in.Description, 0, 2000)
		str("content", in.Content, 0, 50000)
		if in.Category != nil {
			if !categories[*in.Category] {
				v.f["category"] = "unknown category"
			}
			cols["category"] = *in.Category
		}
		if in.Price != nil {
			if *in.Price < 0 || *in.Price > 1000000 {
				v.f["price"] = "must be between 0 and 1000000"
			}
			cols["price"] = fmt.Sprintf("%.2f", *in.Price)
		}
		if in.OldPrice != nil {
			if *in.OldPrice < 0 || *in.OldPrice > 1000000 {
				v.f["old_price"] = "must be between 0 and 1000000"
			}
			if *in.OldPrice == 0 {
				cols["old_price"] = nil
			} else {
				cols["old_price"] = fmt.Sprintf("%.2f", *in.OldPrice)
			}
		}
		if in.Currency != nil {
			if len(*in.Currency) != 3 {
				v.f["currency"] = "must be a 3-letter code"
			}
			cols["currency"] = strings.ToUpper(*in.Currency)
		}
		for k, p := range map[string]*string{"cover_image": in.CoverImage, "demo_url": in.DemoURL, "live_url": in.LiveURL} {
			if p != nil {
				cols[k] = v.url(k, *p, false)
			}
		}
		if in.DownloadURL != nil {
			cols["download_url"] = v.url("download_url", *in.DownloadURL, true)
		}
		if in.DemoType != nil {
			if t := *in.DemoType; t != "" && t != "iframe" && t != "external" && t != "internal" {
				v.f["demo_type"] = "must be iframe, external or internal"
			}
			cols["demo_type"] = *in.DemoType
		}
		if in.MockVariant != nil {
			if *in.MockVariant < 1 || *in.MockVariant > 5 {
				v.f["mock_variant"] = "must be 1-5"
			}
			cols["mock_variant"] = *in.MockVariant
		}
		if in.Featured != nil {
			cols["featured"] = *in.Featured
		}
		if in.Published != nil {
			cols["published"] = *in.Published
		}
		if in.Slug != nil && *in.Slug != "" {
			if !slug.Valid.MatchString(*in.Slug) {
				v.f["slug"] = "lowercase letters, digits and dashes only"
			}
			cols["slug"] = *in.Slug
		}
		if err := v.err(); err != nil {
			return err
		}
		tx, err := s.DB.BeginTx(r.Context(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if !patch {
			if _, ok := cols["slug"]; !ok {
				if cols["slug"], err = uniqueSlug(r.Context(), tx, "products", slug.Make(*in.Title)); err != nil {
					return err
				}
			}
			if id, err = insertRow(r.Context(), tx, "products", cols); err != nil {
				return err
			}
		} else if ok, err := updateRow(r.Context(), tx, "products", id, cols); err != nil {
			return err
		} else if !ok {
			return response.E(404, "PRODUCT_NOT_FOUND", "Product not found.")
		}
		if in.Technologies != nil {
			if err := setList(r.Context(), tx, "product_technologies", "product_id", "name", id, *in.Technologies); err != nil {
				return err
			}
		}
		if in.Features != nil {
			if err := setList(r.Context(), tx, "product_features", "product_id", "text", id, *in.Features); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		act := "product updated"
		if !patch {
			act = "product created"
		}
		s.audit(r, act, "product", id, map[string]any{"fields": len(cols)})
		b, err := s.raw(r.Context(), "SELECT "+productExpr(true)+" FROM products p WHERE p.id=$1", id)
		if err != nil {
			return err
		}
		code := 200
		if !patch {
			code = 201
		}
		response.OK(w, code, b)
		return nil
	}
}

// ---------- projects ----------

func projectExpr() string {
	return `json_build_object('id',p.id,'title',p.title,'slug',p.slug,'category',p.category,'description',p.description,'full_description',p.full_description,
'problem',p.problem,'solution',p.solution,'outcome',p.outcome,'demo_url',p.demo_url,'live_url',p.live_url,'github_url',p.github_url,'cover_image',p.cover_image,
'mock_variant',p.mock_variant,'featured',p.featured,'published',p.published,'created_at',p.created_at,'updated_at',p.updated_at,
'features',` + listSQL("project_features", "text", "project_id") + `,'technologies',` + listSQL("project_technologies", "name", "project_id") + `,'architecture',` + listSQL("project_architecture", "step", "project_id") + `,
'like_count',(SELECT count(*) FROM likes l WHERE l.target_type='project' AND l.target_id=p.id),
'liked',EXISTS(SELECT 1 FROM likes l WHERE l.target_type='project' AND l.target_id=p.id AND l.user_id=$1),
'comment_count',(SELECT count(*) FROM comments m WHERE m.project_id=p.id AND m.status='VISIBLE'))`
}

func (s *Server) listProjects(admin bool) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.URL.Query()
		f := &filter{}
		if !admin {
			f.where = append(f.where, "p.published")
		}
		if q.Get("featured") == "true" {
			f.where = append(f.where, "p.featured")
		}
		if sr := strings.TrimSpace(q.Get("search")); sr != "" {
			f.add("(p.title ILIKE ? OR p.description ILIKE ?)", "%"+escapeLike(sr)+"%")
		}
		return s.paged(w, r, withUser(projectExpr(), uid(r)), "FROM projects p"+f.clause(), sortBy(r, map[string]string{"newest": "p.created_at DESC, p.id DESC"}, "p.position, p.id"), f.args, 50)
	}
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) error {
	cond, arg := refWhere(r.PathValue("ref"), "p", 2)
	return s.one(w, r, "PROJECT_NOT_FOUND", "SELECT "+projectExpr()+" FROM projects p WHERE p.published AND "+cond, uid(r), arg)
}

type projectIn struct {
	Title, Slug, Category, Description, FullDescription, Problem, Solution, Outcome *string
	DemoURL, LiveURL, GithubURL, CoverImage                                         *string
	MockVariant                                                                     *int
	Featured, Published                                                             *bool
	Features, Technologies, Architecture                                            *[]string
}

func (s *Server) saveProject(patch bool) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		var id int64
		if patch {
			var err error
			if id, err = pathID(r, "id"); err != nil {
				return err
			}
		}
		var raw struct {
			Title           *string   `json:"title"`
			Slug            *string   `json:"slug"`
			Category        *string   `json:"category"`
			Description     *string   `json:"description"`
			FullDescription *string   `json:"full_description"`
			Problem         *string   `json:"problem"`
			Solution        *string   `json:"solution"`
			Outcome         *string   `json:"outcome"`
			DemoURL         *string   `json:"demo_url"`
			LiveURL         *string   `json:"live_url"`
			GithubURL       *string   `json:"github_url"`
			CoverImage      *string   `json:"cover_image"`
			MockVariant     *int      `json:"mock_variant"`
			Featured        *bool     `json:"featured"`
			Published       *bool     `json:"published"`
			Features        *[]string `json:"features"`
			Technologies    *[]string `json:"technologies"`
			Architecture    *[]string `json:"architecture"`
		}
		if err := decode(r, &raw); err != nil {
			return err
		}
		v := newV()
		cols := map[string]any{}
		if !patch && raw.Title == nil {
			v.f["title"] = "is required"
		}
		if raw.Title != nil {
			cols["title"] = v.str("title", *raw.Title, 1, 200)
		}
		for k, p := range map[string]*string{"category": raw.Category, "description": raw.Description, "full_description": raw.FullDescription, "problem": raw.Problem, "solution": raw.Solution, "outcome": raw.Outcome} {
			if p != nil {
				cols[k] = v.str(k, *p, 0, 20000)
			}
		}
		for k, p := range map[string]*string{"demo_url": raw.DemoURL, "live_url": raw.LiveURL, "github_url": raw.GithubURL, "cover_image": raw.CoverImage} {
			if p != nil {
				cols[k] = v.url(k, *p, false)
			}
		}
		if raw.MockVariant != nil {
			if *raw.MockVariant < 1 || *raw.MockVariant > 5 {
				v.f["mock_variant"] = "must be 1-5"
			}
			cols["mock_variant"] = *raw.MockVariant
		}
		if raw.Featured != nil {
			cols["featured"] = *raw.Featured
		}
		if raw.Published != nil {
			cols["published"] = *raw.Published
		}
		if raw.Slug != nil && *raw.Slug != "" {
			if !slug.Valid.MatchString(*raw.Slug) {
				v.f["slug"] = "lowercase letters, digits and dashes only"
			}
			cols["slug"] = *raw.Slug
		}
		if err := v.err(); err != nil {
			return err
		}
		tx, err := s.DB.BeginTx(r.Context(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if !patch {
			if _, ok := cols["slug"]; !ok {
				if cols["slug"], err = uniqueSlug(r.Context(), tx, "projects", slug.Make(*raw.Title)); err != nil {
					return err
				}
			}
			cols["position"] = 1000
			if id, err = insertRow(r.Context(), tx, "projects", cols); err != nil {
				return err
			}
		} else if ok, err := updateRow(r.Context(), tx, "projects", id, cols); err != nil {
			return err
		} else if !ok {
			return response.E(404, "PROJECT_NOT_FOUND", "Project not found.")
		}
		for _, l := range []struct {
			tbl, col string
			v        *[]string
		}{{"project_features", "text", raw.Features}, {"project_technologies", "name", raw.Technologies}, {"project_architecture", "step", raw.Architecture}} {
			if l.v != nil {
				if err := setList(r.Context(), tx, l.tbl, "project_id", l.col, id, *l.v); err != nil {
					return err
				}
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		act, code := "project updated", 200
		if !patch {
			act, code = "project created", 201
		}
		s.audit(r, act, "project", id, nil)
		b, err := s.raw(r.Context(), "SELECT "+projectExpr()+" FROM projects p WHERE p.id=$2", uid(r), id)
		if err != nil {
			return err
		}
		response.OK(w, code, b)
		return nil
	}
}

// ---------- blog posts ----------

func postExpr() string {
	return `json_build_object('id',p.id,'title',p.title,'slug',p.slug,'excerpt',p.excerpt,'content',p.content,'cover_image',p.cover_image,'category',c.name,
'reading_time',p.reading_time,'author',p.author_name,'published',p.published,'published_at',p.published_at,'created_at',p.created_at,'updated_at',p.updated_at,
'tags',(SELECT coalesce(json_agg(t.tag ORDER BY t.tag),'[]'::json) FROM post_tags t WHERE t.post_id=p.id),
'like_count',(SELECT count(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id),
'liked',EXISTS(SELECT 1 FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.user_id=$1),
'comment_count',(SELECT count(*) FROM comments m WHERE m.post_id=p.id AND m.status='VISIBLE'))`
}

const postFrom = "FROM posts p JOIN post_categories c ON c.id=p.category_id"

func (s *Server) listPosts(admin bool) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.URL.Query()
		f := &filter{}
		if !admin {
			f.where = append(f.where, "p.published")
		}
		if c := q.Get("category"); c != "" {
			f.add("c.name = ?", c)
		}
		if sr := strings.TrimSpace(q.Get("search")); sr != "" {
			f.add("(p.title ILIKE ? OR p.excerpt ILIKE ?)", "%"+escapeLike(sr)+"%")
		}
		order := sortBy(r, map[string]string{"newest": "coalesce(p.published_at,p.created_at) DESC, p.id DESC", "oldest": "coalesce(p.published_at,p.created_at), p.id", "title": "p.title, p.id"}, "coalesce(p.published_at,p.created_at) DESC, p.id DESC")
		return s.paged(w, r, withUser(postExpr(), uid(r)), postFrom+f.clause(), order, f.args, 12)
	}
}

func (s *Server) getPost(w http.ResponseWriter, r *http.Request) error {
	cond, arg := refWhere(r.PathValue("ref"), "p", 2)
	return s.one(w, r, "POST_NOT_FOUND", "SELECT "+postExpr()+" "+postFrom+" WHERE p.published AND "+cond, uid(r), arg)
}

func (s *Server) savePost(patch bool) hfn {
	return func(w http.ResponseWriter, r *http.Request) error {
		var id int64
		if patch {
			var err error
			if id, err = pathID(r, "id"); err != nil {
				return err
			}
		}
		var in struct {
			Title       *string   `json:"title"`
			Slug        *string   `json:"slug"`
			Excerpt     *string   `json:"excerpt"`
			Content     *string   `json:"content"`
			CoverImage  *string   `json:"cover_image"`
			Category    *string   `json:"category"`
			ReadingTime *string   `json:"reading_time"`
			Author      *string   `json:"author"`
			Published   *bool     `json:"published"`
			Tags        *[]string `json:"tags"`
		}
		if err := decode(r, &in); err != nil {
			return err
		}
		v := newV()
		cols := map[string]any{}
		if !patch {
			if in.Title == nil {
				v.f["title"] = "is required"
			}
			if in.Category == nil {
				v.f["category"] = "is required"
			}
		}
		if in.Title != nil {
			cols["title"] = v.str("title", *in.Title, 1, 250)
		}
		for k, p := range map[string]*string{"excerpt": in.Excerpt, "reading_time": in.ReadingTime, "author_name": in.Author} {
			if p != nil {
				cols[k] = v.str(k, *p, 0, 1000)
			}
		}
		if in.Content != nil {
			cols["content"] = v.str("content", *in.Content, 0, 200000)
		}
		if in.CoverImage != nil {
			cols["cover_image"] = v.url("cover_image", *in.CoverImage, false)
		}
		if in.Slug != nil && *in.Slug != "" {
			if !slug.Valid.MatchString(*in.Slug) {
				v.f["slug"] = "lowercase letters, digits and dashes only"
			}
			cols["slug"] = *in.Slug
		}
		if err := v.err(); err != nil {
			return err
		}
		if in.Category != nil {
			var cid int
			if err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM post_categories WHERE name=$1`, *in.Category).Scan(&cid); err != nil {
				return response.Validation(map[string]string{"category": "unknown category"})
			}
			cols["category_id"] = cid
		}
		if in.ReadingTime == nil && in.Content != nil {
			cols["reading_time"] = fmt.Sprintf("%d min", max(1, (len(strings.Fields(*in.Content))+199)/200))
		}
		tx, err := s.DB.BeginTx(r.Context(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if in.Published != nil {
			cols["published"] = *in.Published
			if *in.Published {
				var has bool
				if patch {
					_ = tx.QueryRowContext(r.Context(), `SELECT published_at IS NOT NULL FROM posts WHERE id=$1`, id).Scan(&has)
				}
				if !has {
					cols["published_at"] = sql.NullTime{Valid: true, Time: nowUTC()}
				}
			}
		}
		if !patch {
			if _, ok := cols["slug"]; !ok {
				if cols["slug"], err = uniqueSlug(r.Context(), tx, "posts", slug.Make(*in.Title)); err != nil {
					return err
				}
			}
			cols["author_id"] = uid(r)
			if _, ok := cols["author_name"]; !ok {
				cols["author_name"] = ctxUser(r).Name
			}
			if id, err = insertRow(r.Context(), tx, "posts", cols); err != nil {
				return err
			}
		} else if ok, err := updateRow(r.Context(), tx, "posts", id, cols); err != nil {
			return err
		} else if !ok {
			return response.E(404, "POST_NOT_FOUND", "Post not found.")
		}
		if in.Tags != nil {
			if _, err := tx.ExecContext(r.Context(), `DELETE FROM post_tags WHERE post_id=$1`, id); err != nil {
				return err
			}
			for _, t := range *in.Tags {
				if t = strings.TrimSpace(t); t != "" && len(t) <= 50 {
					if _, err := tx.ExecContext(r.Context(), `INSERT INTO post_tags (post_id,tag) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, t); err != nil {
						return err
					}
				}
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		act, code := "post updated", 200
		if !patch {
			act, code = "post created", 201
		}
		if in.Published != nil && *in.Published {
			act = "post published"
		}
		s.audit(r, act, "post", id, nil)
		b, err := s.raw(r.Context(), "SELECT "+postExpr()+" "+postFrom+" WHERE p.id=$2", uid(r), id)
		if err != nil {
			return err
		}
		response.OK(w, code, b)
		return nil
	}
}

// withUser inlines the (server-side, integer) user id so list queries can keep their own $n numbering.
func withUser(expr string, uid int64) string {
	return strings.ReplaceAll(expr, "$1", strconv.FormatInt(uid, 10))
}
