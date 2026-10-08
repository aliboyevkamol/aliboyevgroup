// Package api contains the HTTP layer: routing, handlers and SQL access.
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"kamolportfolio/backend/config"
	"kamolportfolio/backend/docs"
	"kamolportfolio/backend/internal/auth"
	"kamolportfolio/backend/internal/payments"
	"kamolportfolio/backend/middleware"
	pkgmail "kamolportfolio/backend/pkg/mail"
	"kamolportfolio/backend/pkg/response"
	"kamolportfolio/backend/pkg/storage"
)

type Server struct {
	DB    *sql.DB
	Cfg   config.Config
	Log   *slog.Logger
	Pay   payments.Provider
	Store *storage.Local
	Mail  pkgmail.Mailer
	authL *middleware.Limiter
}

type hfn func(http.ResponseWriter, *http.Request) error
type ctxKey int

const userKey ctxKey = 1

type User struct {
	ID                int64
	Name, Email, Role string
}

const (
	Public = iota
	Authed
	AdminOnly
)

func ctxUser(r *http.Request) *User { u, _ := r.Context().Value(userKey).(*User); return u }
func uid(r *http.Request) int64 {
	if u := ctxUser(r); u != nil {
		return u.ID
	}
	return 0
}

// h converts handler errors into the standard error envelope. Internal errors are never leaked.
func (s *Server) h(f hfn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			var e *response.Error
			if errors.As(err, &e) {
				response.Fail(w, e)
				return
			}
			var pe *pq.Error
			if errors.As(err, &pe) {
				switch pe.Code {
				case "23505":
					response.Fail(w, response.E(409, "ALREADY_EXISTS", "This value already exists."))
					return
				case "23514", "22P02", "23502", "22001":
					response.Fail(w, response.E(422, "VALIDATION_ERROR", "Invalid value."))
					return
				case "23503":
					response.Fail(w, response.E(422, "INVALID_REFERENCE", "Referenced item does not exist."))
					return
				}
			}
			s.Log.Error("unhandled error", "err", err.Error(), "path", r.URL.Path, "request_id", middleware.RID(r))
			response.Fail(w, response.E(500, "INTERNAL_ERROR", "Something went wrong. Please try again."))
		}
	}
}

// route wraps a handler with the authentication level it needs.
func (s *Server) route(level int, f hfn) http.Handler {
	return s.h(func(w http.ResponseWriter, r *http.Request) error {
		var u *User
		if tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
			c, err := auth.Parse(s.Cfg.JWTSecret, tok)
			if err == nil {
				var status string
				u = &User{}
				err = s.DB.QueryRowContext(r.Context(), `SELECT id,name,email,role,status FROM users WHERE id=$1`, c.Sub).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &status)
				if err != nil || status != "ACTIVE" {
					u = nil
					if err == nil && level > Public {
						return response.E(403, "ACCOUNT_BLOCKED", "This account is not active.")
					}
				}
			}
			if u == nil && level > Public {
				return response.E(401, "UNAUTHORIZED", "Please log in to continue.")
			}
		} else if level > Public {
			return response.E(401, "UNAUTHORIZED", "Please log in to continue.")
		}
		if level == AdminOnly && (u == nil || u.Role != "ADMIN") {
			return response.E(403, "FORBIDDEN", "You do not have permission to do this.")
		}
		if u != nil {
			r = r.WithContext(context.WithValue(r.Context(), userKey, u))
		}
		return f(w, r)
	})
}

// ---------- helpers ----------

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return response.E(400, "BAD_JSON", "Invalid request body.")
	}
	return nil
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id < 1 {
		return 0, response.E(400, "INVALID_ID", "Invalid id.")
	}
	return id, nil
}

func qint(r *http.Request, k string, def, min, max int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(k))
	if err != nil || n < min {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func (s *Server) raw(ctx context.Context, q string, args ...any) (json.RawMessage, error) {
	var b []byte
	if err := s.DB.QueryRowContext(ctx, q, args...).Scan(&b); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, sql.ErrNoRows
	}
	return b, nil
}

func (s *Server) one(w http.ResponseWriter, r *http.Request, notFound string, q string, args ...any) error {
	b, err := s.raw(r.Context(), q, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return response.E(404, notFound, "Not found.")
	}
	if err != nil {
		return err
	}
	response.OK(w, 200, b)
	return nil
}

// paged runs "SELECT <expr> <from>" with pagination and returns the list envelope.
func (s *Server) paged(w http.ResponseWriter, r *http.Request, expr, from, order string, args []any, defLimit int) error {
	page, limit := qint(r, "page", 1, 1, 100000), qint(r, "limit", defLimit, 1, 100)
	var total int
	if err := s.DB.QueryRowContext(r.Context(), "SELECT count(*) "+from, args...).Scan(&total); err != nil {
		return err
	}
	q := fmt.Sprintf("SELECT coalesce(json_agg(x.j),'[]'::json) FROM (SELECT %s AS j %s ORDER BY %s LIMIT %d OFFSET %d) x", expr, from, order, limit, (page-1)*limit)
	b, err := s.raw(r.Context(), q, args...)
	if err != nil {
		return err
	}
	pages := (total + limit - 1) / limit
	response.List(w, b, response.Pagination{Page: page, Limit: limit, Total: total, Pages: pages})
	return nil
}

type V struct{ f map[string]string }

func newV() *V { return &V{f: map[string]string{}} }
func (v *V) err() error {
	if len(v.f) > 0 {
		return response.Validation(v.f)
	}
	return nil
}
func (v *V) str(field, val string, min, max int) string {
	val = strings.TrimSpace(val)
	if n := len([]rune(val)); n < min || n > max {
		v.f[field] = fmt.Sprintf("must be %d-%d characters", min, max)
	}
	return val
}
func (v *V) email(field, val string) string {
	val = strings.TrimSpace(val)
	if a, err := mail.ParseAddress(val); err != nil || a.Address != val || len(val) > 254 {
		v.f[field] = "must be a valid email address"
	}
	return strings.ToLower(val)
}
func (v *V) url(field, val string, allowKey bool) string {
	val = strings.TrimSpace(val)
	if val == "" || (allowKey && !strings.Contains(val, "://") && !strings.Contains(val, "..")) {
		return val
	}
	if !(strings.HasPrefix(val, "https://") || strings.HasPrefix(val, "http://")) || len(val) > 1000 {
		v.f[field] = "must be an http(s) URL"
	}
	return val
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (s *Server) audit(r *http.Request, action, entity string, id any, meta map[string]any) {
	b, _ := json.Marshal(meta)
	if meta == nil {
		b = []byte("{}")
	}
	var u any
	if cu := ctxUser(r); cu != nil {
		u = cu.ID
	}
	_, err := s.DB.ExecContext(context.Background(), `INSERT INTO audit_logs (user_id,action,entity_type,entity_id,metadata) VALUES ($1,$2,$3,$4,$5)`, u, action, entity, fmt.Sprint(id), string(b))
	if err != nil {
		s.Log.Error("audit failed", "err", err.Error())
	}
}

// ---------- routes ----------

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	R := func(pattern string, level int, f hfn) { mux.Handle(pattern, s.route(level, f)) }
	const v = "/api/v1"

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { response.OKRaw(w, `{"status":"ok"}`) })
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.DB.PingContext(r.Context()); err != nil {
			response.Fail(w, response.E(503, "NOT_READY", "Database is not reachable."))
			return
		}
		response.OKRaw(w, `{"status":"ready"}`)
	})
	mux.HandleFunc("GET "+v+"/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(docs.OpenAPI)
	})
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'self' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net")
		_, _ = w.Write([]byte(docs.SwaggerHTML))
	})
	mux.HandleFunc("GET /files/{key...}", s.serveFile)

	// auth
	R("POST "+v+"/auth/register", Public, s.register)
	R("POST "+v+"/auth/login", Public, s.login)
	R("POST "+v+"/auth/logout", Public, s.logout)
	R("POST "+v+"/auth/refresh", Public, s.refresh)
	R("GET "+v+"/auth/me", Authed, s.me)
	R("POST "+v+"/auth/forgot-password", Public, s.forgot)
	R("POST "+v+"/auth/reset-password", Public, s.reset)
	// profile
	R("GET "+v+"/me", Authed, s.me)
	R("PATCH "+v+"/me", Authed, s.updateMe)
	R("GET "+v+"/me/orders", Authed, s.myOrders)
	R("GET "+v+"/me/purchases", Authed, s.myPurchases)
	R("GET "+v+"/me/favorites", Authed, s.myFavorites)
	// catalog
	R("GET "+v+"/products", Public, s.listProducts(false))
	R("GET "+v+"/products/{ref}", Public, s.getProduct)
	R("GET "+v+"/products/{id}/access", Authed, s.productAccess)
	R("GET "+v+"/products/{id}/download", Authed, s.productDownload)
	R("POST "+v+"/products/{id}/favorite", Authed, s.favorite(true))
	R("DELETE "+v+"/products/{id}/favorite", Authed, s.favorite(false))
	R("GET "+v+"/projects", Public, s.listProjects(false))
	R("GET "+v+"/projects/{ref}", Public, s.getProject)
	R("GET "+v+"/posts", Public, s.listPosts(false))
	R("GET "+v+"/posts/{ref}", Public, s.getPost)
	R("GET "+v+"/portfolio", Public, s.portfolio)
	R("POST "+v+"/contact", Public, s.contact)
	// social
	R("GET "+v+"/posts/{id}/comments", Public, s.listComments("post"))
	R("POST "+v+"/posts/{id}/comments", Authed, s.addComment("post"))
	R("GET "+v+"/projects/{id}/comments", Public, s.listComments("project"))
	R("POST "+v+"/projects/{id}/comments", Authed, s.addComment("project"))
	R("POST "+v+"/comments/{id}/reply", Authed, s.reply)
	R("PATCH "+v+"/comments/{id}", Authed, s.editComment)
	R("DELETE "+v+"/comments/{id}", Authed, s.deleteComment)
	for _, t := range []string{"posts", "projects", "comments"} {
		R("POST "+v+"/"+t+"/{id}/like", Authed, s.like(t, true))
		R("DELETE "+v+"/"+t+"/{id}/like", Authed, s.like(t, false))
	}
	// cart, orders, payments
	R("GET "+v+"/cart", Authed, s.getCart)
	R("POST "+v+"/cart/items", Authed, s.addCartItem)
	R("PATCH "+v+"/cart/items/{product_id}", Authed, s.setCartItem)
	R("DELETE "+v+"/cart/items/{product_id}", Authed, s.removeCartItem)
	R("DELETE "+v+"/cart", Authed, s.clearCart)
	R("POST "+v+"/orders", Authed, s.createOrder)
	R("GET "+v+"/orders/{id}", Authed, s.getOrder)
	R("POST "+v+"/payments", Authed, s.createPayment)
	R("GET "+v+"/payments/{id}", Authed, s.getPayment)
	R("POST "+v+"/payments/webhook", Public, s.webhook)
	R("POST "+v+"/payments/test/complete", Authed, s.testComplete)
	// admin
	R("GET "+v+"/admin/dashboard", AdminOnly, s.dashboard)
	R("GET "+v+"/admin/products", AdminOnly, s.listProducts(true))
	R("POST "+v+"/admin/products", AdminOnly, s.saveProduct(false))
	R("GET "+v+"/admin/products/{id}", AdminOnly, s.adminGetProduct)
	R("PATCH "+v+"/admin/products/{id}", AdminOnly, s.saveProduct(true))
	R("DELETE "+v+"/admin/products/{id}", AdminOnly, s.adminDelete("products", "product"))
	R("GET "+v+"/admin/projects", AdminOnly, s.listProjects(true))
	R("POST "+v+"/admin/projects", AdminOnly, s.saveProject(false))
	R("PATCH "+v+"/admin/projects/{id}", AdminOnly, s.saveProject(true))
	R("DELETE "+v+"/admin/projects/{id}", AdminOnly, s.adminDelete("projects", "project"))
	R("GET "+v+"/admin/posts", AdminOnly, s.listPosts(true))
	R("POST "+v+"/admin/posts", AdminOnly, s.savePost(false))
	R("PATCH "+v+"/admin/posts/{id}", AdminOnly, s.savePost(true))
	R("DELETE "+v+"/admin/posts/{id}", AdminOnly, s.adminDelete("posts", "post"))
	R("GET "+v+"/admin/orders", AdminOnly, s.adminOrders)
	R("GET "+v+"/admin/orders/{id}", AdminOnly, s.adminGetOrder)
	R("PATCH "+v+"/admin/orders/{id}", AdminOnly, s.adminSetOrder)
	R("GET "+v+"/admin/users", AdminOnly, s.adminUsers)
	R("PATCH "+v+"/admin/users/{id}", AdminOnly, s.adminSetUser)
	R("GET "+v+"/admin/comments", AdminOnly, s.adminComments)
	R("PATCH "+v+"/admin/comments/{id}", AdminOnly, s.adminSetComment)
	R("DELETE "+v+"/admin/comments/{id}", AdminOnly, s.adminDeleteComment)
	R("GET "+v+"/admin/portfolio", AdminOnly, s.portfolio)
	R("PUT "+v+"/admin/portfolio", AdminOnly, s.savePortfolio)
	R("GET "+v+"/admin/media", AdminOnly, s.listMedia)
	R("POST "+v+"/admin/media", AdminOnly, s.uploadMedia)
	R("DELETE "+v+"/admin/media/{id}", AdminOnly, s.deleteMedia)
	R("GET "+v+"/admin/contact-messages", AdminOnly, s.listContact)
	R("PATCH "+v+"/admin/contact-messages/{id}", AdminOnly, s.setContact)

	// Rate limits: strict for credential/contact endpoints, general for the rest of the API.
	general := middleware.NewLimiter(s.Cfg.APIRate, s.Cfg.APIBurst, s.Cfg.TrustProxy)
	s.authL = middleware.NewLimiter(s.Cfg.AuthRate, s.Cfg.AuthBurst, s.Cfg.TrustProxy)
	strict, loose := s.authL.Middleware(mux), general.Middleware(mux)
	static := http.NotFoundHandler()
	if s.Cfg.StaticDir != "" {
		static = staticHandler(s.Cfg.StaticDir)
	}
	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strictPaths[p]:
			strict.ServeHTTP(w, r)
		case p == "/api/v1/payments/webhook":
			mux.ServeHTTP(w, r)
		case strings.HasPrefix(p, "/api/"):
			loose.ServeHTTP(w, r)
		case p == "/health" || p == "/ready" || p == "/docs" || strings.HasPrefix(p, "/files/"):
			mux.ServeHTTP(w, r)
		default:
			static.ServeHTTP(w, r)
		}
	})
	return middleware.RequestID(middleware.Logger(s.Log)(middleware.Recover(s.Log)(middleware.Security(s.Cfg.IsProd())(middleware.CORS(s.Cfg.Origins())(root)))))
}

// Credential-guessing and spam targets get the strict limiter; refresh/me are called on every page load.
var strictPaths = map[string]bool{"/api/v1/auth/login": true, "/api/v1/auth/register": true, "/api/v1/auth/forgot-password": true, "/api/v1/auth/reset-password": true, "/api/v1/contact": true}

var staticExt = map[string]bool{".html": true, ".css": true, ".js": true, ".png": true, ".jpg": true, ".jpeg": true, ".svg": true, ".webp": true, ".ico": true, ".woff2": true}

// staticHandler serves the frontend. It only serves whitelisted file types and never hidden files or the backend folder.
func staticHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean(r.URL.Path)
		for _, seg := range strings.Split(p, "/") {
			if strings.HasPrefix(seg, ".") && seg != "" || seg == "backend" || seg == "node_modules" {
				http.NotFound(w, r)
				return
			}
		}
		if ext := filepath.Ext(p); p != "/" && !strings.HasSuffix(r.URL.Path, "/") && !staticExt[ext] {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})
}
