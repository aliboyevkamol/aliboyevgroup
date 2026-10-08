package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"kamolportfolio/backend/internal/auth"
	"kamolportfolio/backend/pkg/response"
)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 30 * 24 * time.Hour
	cookieName = "kamol_rt"
)

// rotateGrace lets two parallel page loads share one refresh token without being treated as token theft.
var rotateGrace = 10 * time.Second

func hashToken(secret, tok string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(tok))
	return hex.EncodeToString(m.Sum(nil))
}

type userRow struct {
	ID                                int64
	Name, Email, Avatar, Role, Status string
}

func (u userRow) public() map[string]any {
	return map[string]any{"id": u.ID, "name": u.Name, "email": u.Email, "avatar_url": u.Avatar, "role": u.Role, "status": u.Status}
}

// issue creates an access token and a rotating refresh token (cookie) for the user.
func (s *Server) issue(w http.ResponseWriter, r *http.Request, u userRow, family string) error {
	access, err := auth.Sign(s.Cfg.JWTSecret, u.ID, u.Role, accessTTL)
	if err != nil {
		return err
	}
	if family == "" {
		family = auth.RandomToken(12)
	}
	rt := auth.RandomToken(32)
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO refresh_tokens (user_id,family,token_hash,expires_at) VALUES ($1,$2,$3,$4)`, u.ID, family, hashToken(s.Cfg.RefreshSecret, rt), time.Now().Add(refreshTTL)); err != nil {
		return err
	}
	s.setCookie(w, rt, int(refreshTTL.Seconds()))
	return nil_or(w, access, u)
}

func nil_or(w http.ResponseWriter, access string, u userRow) error {
	response.OK(w, 200, map[string]any{"access_token": access, "expires_in": int(accessTTL.Seconds()), "user": u.public()})
	return nil
}

func (s *Server) setCookie(w http.ResponseWriter, val string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: val, Path: "/api/v1/auth", MaxAge: maxAge, HttpOnly: true, Secure: s.Cfg.CookieSecure(), SameSite: s.Cfg.CookieSameSite()})
}

func (s *Server) checkOrigin(r *http.Request) error {
	o := r.Header.Get("Origin")
	if o == "" {
		return nil
	}
	for _, a := range s.Cfg.Origins() {
		if a == o {
			return nil
		}
	}
	if o == "http://"+r.Host || o == "https://"+r.Host {
		return nil
	}
	return response.E(403, "BAD_ORIGIN", "Request origin is not allowed.")
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	v := newV()
	in.Name = v.str("name", in.Name, 1, 100)
	in.Email = v.email("email", in.Email)
	if n := len(in.Password); n < 8 || n > 128 {
		v.f["password"] = "must be 8-128 characters"
	}
	if in.Password != in.ConfirmPassword {
		v.f["confirm_password"] = "passwords do not match"
	}
	if err := v.err(); err != nil {
		return err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}
	u := userRow{Name: in.Name, Email: in.Email, Role: "USER", Status: "ACTIVE"}
	err = s.DB.QueryRowContext(r.Context(), `INSERT INTO users (name,email,password_hash,last_login_at) VALUES ($1,$2,$3,now()) RETURNING id`, in.Name, in.Email, hash).Scan(&u.ID)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return response.E(409, "EMAIL_TAKEN", "An account with this email already exists.")
		}
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	return s.issue(w, r, u, "")
}

var dummyHash, _ = auth.HashPassword("dummy-password-for-timing")

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Email, Password string }
	if err := decode(r, &in); err != nil {
		return err
	}
	var u userRow
	var hash string
	err := s.DB.QueryRowContext(r.Context(), `SELECT id,name,email,avatar_url,role,status,password_hash FROM users WHERE lower(email)=lower($1)`, strings.TrimSpace(in.Email)).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar, &u.Role, &u.Status, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		auth.VerifyPassword(in.Password, dummyHash) // equalize timing
		return response.E(401, "INVALID_CREDENTIALS", "Incorrect email or password.")
	}
	if err != nil {
		return err
	}
	if !auth.VerifyPassword(in.Password, hash) {
		return response.E(401, "INVALID_CREDENTIALS", "Incorrect email or password.")
	}
	if u.Status != "ACTIVE" {
		return response.E(403, "ACCOUNT_BLOCKED", "This account is not active.")
	}
	_, _ = s.DB.ExecContext(r.Context(), `UPDATE users SET last_login_at=now() WHERE id=$1`, u.ID)
	return s.issue(w, r, u, "")
}

// refresh rotates the refresh token. Re-use of an already rotated token (outside a short grace window) revokes the whole family.
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) error {
	if err := s.checkOrigin(r); err != nil {
		return err
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return response.E(401, "NO_REFRESH_TOKEN", "Please log in again.")
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id, userID int64
	var family string
	var expires time.Time
	var rotated, revoked sql.NullTime
	err = tx.QueryRowContext(r.Context(), `SELECT id,user_id,family,expires_at,rotated_at,revoked_at FROM refresh_tokens WHERE token_hash=$1 FOR UPDATE`, hashToken(s.Cfg.RefreshSecret, c.Value)).Scan(&id, &userID, &family, &expires, &rotated, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		s.setCookie(w, "", -1)
		return response.E(401, "INVALID_REFRESH_TOKEN", "Please log in again.")
	}
	if err != nil {
		return err
	}
	if revoked.Valid || time.Now().After(expires) {
		s.setCookie(w, "", -1)
		return response.E(401, "INVALID_REFRESH_TOKEN", "Please log in again.")
	}
	if rotated.Valid {
		if time.Since(rotated.Time) > rotateGrace {
			_, _ = tx.ExecContext(r.Context(), `UPDATE refresh_tokens SET revoked_at=now() WHERE family=$1 AND revoked_at IS NULL`, family)
			_ = tx.Commit()
			s.setCookie(w, "", -1)
			return response.E(401, "TOKEN_REUSE", "Session expired. Please log in again.")
		}
	} else if _, err := tx.ExecContext(r.Context(), `UPDATE refresh_tokens SET rotated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	var u userRow
	if err := tx.QueryRowContext(r.Context(), `SELECT id,name,email,avatar_url,role,status FROM users WHERE id=$1`, userID).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar, &u.Role, &u.Status); err != nil {
		return err
	}
	if u.Status != "ACTIVE" {
		return response.E(403, "ACCOUNT_BLOCKED", "This account is not active.")
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.issue(w, r, u, family)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	if err := s.checkOrigin(r); err != nil {
		return err
	}
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		_, _ = s.DB.ExecContext(r.Context(), `UPDATE refresh_tokens SET revoked_at=now() WHERE family=(SELECT family FROM refresh_tokens WHERE token_hash=$1) AND revoked_at IS NULL`, hashToken(s.Cfg.RefreshSecret, c.Value))
	}
	s.setCookie(w, "", -1)
	response.OK(w, 200, map[string]bool{"logged_out": true})
	return nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	return s.one(w, r, "USER_NOT_FOUND", `SELECT json_build_object('id',id,'name',name,'email',email,'avatar_url',avatar_url,'role',role,'status',status,'created_at',created_at,'last_login_at',last_login_at) FROM users WHERE id=$1`, uid(r))
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Name            *string `json:"name"`
		AvatarURL       *string `json:"avatar_url"`
		CurrentPassword string  `json:"current_password"`
		NewPassword     string  `json:"new_password"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	v := newV()
	if in.Name != nil {
		*in.Name = v.str("name", *in.Name, 1, 100)
	}
	if in.AvatarURL != nil {
		*in.AvatarURL = v.url("avatar_url", *in.AvatarURL, false)
	}
	if in.NewPassword != "" && (len(in.NewPassword) < 8 || len(in.NewPassword) > 128) {
		v.f["new_password"] = "must be 8-128 characters"
	}
	if err := v.err(); err != nil {
		return err
	}
	if in.NewPassword != "" {
		var h string
		if err := s.DB.QueryRowContext(r.Context(), `SELECT password_hash FROM users WHERE id=$1`, uid(r)).Scan(&h); err != nil {
			return err
		}
		if !auth.VerifyPassword(in.CurrentPassword, h) {
			return response.E(403, "WRONG_PASSWORD", "Current password is incorrect.")
		}
		nh, err := auth.HashPassword(in.NewPassword)
		if err != nil {
			return err
		}
		if _, err := s.DB.ExecContext(r.Context(), `UPDATE users SET password_hash=$1, updated_at=now() WHERE id=$2`, nh, uid(r)); err != nil {
			return err
		}
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE users SET name=coalesce($1,name), avatar_url=coalesce($2,avatar_url), updated_at=now() WHERE id=$3`, in.Name, in.AvatarURL, uid(r)); err != nil {
		return err
	}
	return s.me(w, r)
}

func (s *Server) forgot(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Email string }
	if err := decode(r, &in); err != nil {
		return err
	}
	var id int64
	err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM users WHERE lower(email)=lower($1) AND status='ACTIVE'`, strings.TrimSpace(in.Email)).Scan(&id)
	if err == nil {
		tok := auth.RandomToken(32)
		if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO password_resets (user_id,token_hash,expires_at) VALUES ($1,$2,now()+interval '1 hour')`, id, hashToken(s.Cfg.RefreshSecret, tok)); err != nil {
			return err
		}
		base := s.Cfg.PublicURL
		if o := s.Cfg.Origins(); len(o) > 0 {
			base = o[0]
		}
		go func(to, link string) {
			_ = s.Mail.Send(to, "Reset your password", "Use this link within one hour to reset your password:\n"+link)
		}(strings.TrimSpace(in.Email), base+"/login.html?reset="+tok)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	response.OK(w, 200, map[string]string{"message": "If an account exists for this email, a reset link has been sent."})
	return nil
}

func (s *Server) reset(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Token, Password string }
	if err := decode(r, &in); err != nil {
		return err
	}
	if n := len(in.Password); n < 8 || n > 128 {
		return response.Validation(map[string]string{"password": "must be 8-128 characters"})
	}
	h, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var uidv int64
	err = tx.QueryRowContext(r.Context(), `UPDATE password_resets SET used_at=now() WHERE token_hash=$1 AND used_at IS NULL AND expires_at>now() RETURNING user_id`, hashToken(s.Cfg.RefreshSecret, in.Token)).Scan(&uidv)
	if errors.Is(err, sql.ErrNoRows) {
		return response.E(400, "INVALID_RESET_TOKEN", "This reset link is invalid or has expired.")
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE users SET password_hash=$1, updated_at=now() WHERE id=$2`, h, uidv); err != nil {
		return err
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE refresh_tokens SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, uidv); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	response.OK(w, 200, map[string]string{"message": "Password updated. Please log in."})
	return nil
}
