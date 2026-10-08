package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"kamolportfolio/backend/config"
	"kamolportfolio/backend/database"
	"kamolportfolio/backend/internal/auth"
	"kamolportfolio/backend/internal/payments"
	"kamolportfolio/backend/pkg/mail"
	"kamolportfolio/backend/pkg/storage"
)

type env struct {
	ts  *httptest.Server
	srv *Server
	db  *sql.DB
}

var E *env

func newServer(db *sql.DB, mod func(*config.Config)) *env {
	cfg := config.Config{Env: "test", Port: "0", JWTSecret: "test-jwt-secret", RefreshSecret: "test-refresh-secret", FrontendURL: "http://localhost:5500", WebhookSecret: "test-webhook-secret",
		PaymentProvider: "test", PublicURL: "http://localhost", AuthRate: 100000, AuthBurst: 100000, APIRate: 100000, APIBurst: 100000}
	if mod != nil {
		mod(&cfg)
	}
	var lw io.Writer = io.Discard
	if os.Getenv("TEST_LOG") != "" {
		lw = os.Stderr
	}
	log := slog.New(slog.NewTextHandler(lw, nil))
	st, _ := storage.NewLocal(os.TempDir()+"/kamol-test-uploads", "http://localhost", "test-storage-secret")
	s := &Server{DB: db, Cfg: cfg, Log: log, Pay: &payments.Test{Secret: cfg.WebhookSecret}, Store: st, Mail: mail.Log{L: log}}
	return &env{ts: httptest.NewServer(s.Routes()), srv: s, db: db}
}

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		fmt.Println("TEST_DATABASE_URL not set: skipping API tests")
		os.Exit(0)
	}
	auth.Iterations = 1000
	rotateGrace = 0
	db, err := sql.Open("postgres", url)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		panic(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := database.Migrate(ctx, db, log); err != nil {
		panic(err)
	}
	if err := database.Seed(ctx, db, log); err != nil {
		panic(err)
	}
	h, _ := auth.HashPassword("AdminPassw0rd!")
	db.ExecContext(ctx, `INSERT INTO users (name,email,password_hash,role) VALUES ('Admin','admin@test.dev',$1,'ADMIN')`, h)
	db.ExecContext(ctx, `INSERT INTO coupons (code,percent) VALUES ('HALF',50)`)
	E = newServer(db, nil)
	os.Exit(m.Run())
}

type res struct {
	Code int
	Body map[string]any
	H    http.Header
	C    []*http.Cookie
}

func (e *env) do(t *testing.T, method, path string, body any, tok string, hdr ...string) res {
	t.Helper()
	var rd io.Reader
	if s, ok := body.(string); ok {
		rd = strings.NewReader(s)
	} else if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.ts.URL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	return res{resp.StatusCode, m, resp.Header, resp.Cookies()}
}

func dig(m map[string]any, path ...string) any {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[p]
	}
	return cur
}
func num(v any) int    { f, _ := v.(float64); return int(f) }
func str(v any) string { s, _ := v.(string); return s }
func want(t *testing.T, r res, code int) {
	t.Helper()
	if r.Code != code {
		t.Fatalf("expected HTTP %d, got %d: %v", code, r.Code, r.Body)
	}
}

func register(t *testing.T, email string) (tok string, id int) {
	t.Helper()
	r := E.do(t, "POST", "/api/v1/auth/register", map[string]string{"name": "Test User", "email": email, "password": "Passw0rd!x", "confirm_password": "Passw0rd!x"}, "")
	want(t, r, 200)
	return str(dig(r.Body, "data", "access_token")), num(dig(r.Body, "data", "user", "id"))
}
func adminTok(t *testing.T) string {
	r := E.do(t, "POST", "/api/v1/auth/login", map[string]string{"email": "admin@test.dev", "password": "AdminPassw0rd!"}, "")
	want(t, r, 200)
	return str(dig(r.Body, "data", "access_token"))
}

func TestHealthAndEnvelope(t *testing.T) {
	want(t, E.do(t, "GET", "/health", nil, ""), 200)
	want(t, E.do(t, "GET", "/ready", nil, ""), 200)
	r := E.do(t, "GET", "/api/v1/products/does-not-exist", nil, "")
	want(t, r, 404)
	if r.Body["success"] != false || dig(r.Body, "error", "code") != "PRODUCT_NOT_FOUND" {
		t.Fatalf("bad error envelope: %v", r.Body)
	}
	if r.H.Get("X-Content-Type-Options") != "nosniff" || r.H.Get("X-Request-ID") == "" {
		t.Fatal("missing security/request-id headers")
	}
}

func TestAuth(t *testing.T) {
	r := E.do(t, "POST", "/api/v1/auth/register", map[string]string{"name": "", "email": "bad", "password": "short", "confirm_password": "x"}, "")
	want(t, r, 422)
	tok, _ := register(t, "dup@test.dev")
	want(t, E.do(t, "POST", "/api/v1/auth/register", map[string]string{"name": "X", "email": "DUP@test.dev", "password": "Passw0rd!x", "confirm_password": "Passw0rd!x"}, ""), 409)
	want(t, E.do(t, "POST", "/api/v1/auth/login", map[string]string{"email": "dup@test.dev", "password": "wrong-password"}, ""), 401)
	want(t, E.do(t, "POST", "/api/v1/auth/login", map[string]string{"email": "nobody@test.dev", "password": "wrong-password"}, ""), 401)
	want(t, E.do(t, "GET", "/api/v1/auth/me", nil, ""), 401)
	want(t, E.do(t, "GET", "/api/v1/auth/me", nil, "garbage.token.value"), 401)
	me := E.do(t, "GET", "/api/v1/me", nil, tok)
	want(t, me, 200)
	if _, leaked := me.Body["data"].(map[string]any)["password_hash"]; leaked {
		t.Fatal("password_hash leaked")
	}
	// refresh token rotation and reuse detection
	l := E.do(t, "POST", "/api/v1/auth/login", map[string]string{"email": "dup@test.dev", "password": "Passw0rd!x"}, "")
	want(t, l, 200)
	var cookie *http.Cookie
	for _, c := range l.C {
		if c.Name == cookieName {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly {
		t.Fatal("refresh cookie missing or not HttpOnly")
	}
	rf := func(c *http.Cookie) res {
		req, _ := http.NewRequest("POST", E.ts.URL+"/api/v1/auth/refresh", nil)
		req.AddCookie(c)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var m map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&m)
		return res{resp.StatusCode, m, resp.Header, resp.Cookies()}
	}
	r1 := rf(cookie)
	want(t, r1, 200)
	want(t, rf(cookie), 401) // old token re-used -> family revoked
	var newCookie *http.Cookie
	for _, c := range r1.C {
		if c.Name == cookieName {
			newCookie = c
		}
	}
	want(t, rf(newCookie), 401) // whole family is revoked after reuse
	// bad origin
	req, _ := http.NewRequest("POST", E.ts.URL+"/api/v1/auth/refresh", nil)
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 for foreign origin, got %d", resp.StatusCode)
	}
	// forgot password never reveals whether the email exists
	want(t, E.do(t, "POST", "/api/v1/auth/forgot-password", map[string]string{"email": "nobody@test.dev"}, ""), 200)
	want(t, E.do(t, "POST", "/api/v1/auth/reset-password", map[string]string{"token": "nope", "password": "Passw0rd!x"}, ""), 400)
}

func TestBlockedUserAndAdminAuthorization(t *testing.T) {
	tok, id := register(t, "blockme@test.dev")
	for _, p := range []string{"/api/v1/admin/dashboard", "/api/v1/admin/products", "/api/v1/admin/users", "/api/v1/admin/orders", "/api/v1/admin/comments"} {
		want(t, E.do(t, "GET", p, nil, tok), 403)
		want(t, E.do(t, "GET", p, nil, ""), 401)
	}
	at := adminTok(t)
	want(t, E.do(t, "GET", "/api/v1/admin/dashboard", nil, at), 200)
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/users/%d", id), map[string]string{"status": "BLOCKED"}, at), 200)
	want(t, E.do(t, "GET", "/api/v1/me", nil, tok), 403) // existing token stops working
	want(t, E.do(t, "POST", "/api/v1/auth/login", map[string]string{"email": "blockme@test.dev", "password": "Passw0rd!x"}, ""), 403)
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/users/%d", id), map[string]string{"status": "ACTIVE"}, at), 200)
	want(t, E.do(t, "POST", "/api/v1/auth/login", map[string]string{"email": "blockme@test.dev", "password": "Passw0rd!x"}, ""), 200)
}

func TestProductsPublic(t *testing.T) {
	r := E.do(t, "GET", "/api/v1/products?limit=5&page=1", nil, "")
	want(t, r, 200)
	if num(dig(r.Body, "pagination", "total")) != 8 || len(r.Body["data"].([]any)) != 5 || num(dig(r.Body, "pagination", "pages")) != 2 {
		t.Fatalf("pagination wrong: %v", r.Body["pagination"])
	}
	if _, ok := r.Body["data"].([]any)[0].(map[string]any)["download_url"]; ok {
		t.Fatal("public product leaks download_url")
	}
	r = E.do(t, "GET", "/api/v1/products?category=saas", nil, "")
	for _, p := range r.Body["data"].([]any) {
		if str(p.(map[string]any)["category"]) != "saas" {
			t.Fatal("category filter broken")
		}
	}
	r = E.do(t, "GET", "/api/v1/products?search=dashboard", nil, "")
	if num(dig(r.Body, "pagination", "total")) < 1 {
		t.Fatal("search found nothing")
	}
	r = E.do(t, "GET", "/api/v1/products?sort=price_asc&limit=100", nil, "")
	prev := -1.0
	for _, p := range r.Body["data"].([]any) {
		pr := p.(map[string]any)["price"].(float64)
		if pr < prev {
			t.Fatal("price_asc not sorted")
		}
		prev = pr
	}
	want(t, E.do(t, "GET", "/api/v1/products?category=bogus", nil, ""), 422)
	want(t, E.do(t, "GET", "/api/v1/products/saas-starter", nil, ""), 200)
	want(t, E.do(t, "GET", "/api/v1/products/1", nil, ""), 200)
}

func TestProjectsBlogSocial(t *testing.T) {
	r := E.do(t, "GET", "/api/v1/projects", nil, "")
	want(t, r, 200)
	if len(r.Body["data"].([]any)) != 5 {
		t.Fatal("expected 5 seeded projects")
	}
	want(t, E.do(t, "GET", "/api/v1/projects/offpay", nil, ""), 200)
	want(t, E.do(t, "GET", "/api/v1/posts?category=SaaS", nil, ""), 200)
	pr := E.do(t, "GET", "/api/v1/projects/offpay", nil, "")
	pid := num(dig(pr.Body, "data", "id"))
	tok, _ := register(t, "social@test.dev")
	// likes: duplicate-safe
	for i := 0; i < 3; i++ {
		want(t, E.do(t, "POST", fmt.Sprintf("/api/v1/projects/%d/like", pid), nil, tok), 200)
	}
	l := E.do(t, "GET", "/api/v1/projects/offpay", nil, tok)
	if num(dig(l.Body, "data", "like_count")) != 1 || dig(l.Body, "data", "liked") != true {
		t.Fatalf("duplicate like created: %v", l.Body["data"])
	}
	want(t, E.do(t, "POST", fmt.Sprintf("/api/v1/projects/%d/like", pid), nil, ""), 401)
	want(t, E.do(t, "DELETE", fmt.Sprintf("/api/v1/projects/%d/like", pid), nil, tok), 200)
	want(t, E.do(t, "POST", "/api/v1/posts/1/like", nil, tok), 200)
	// comments
	want(t, E.do(t, "POST", fmt.Sprintf("/api/v1/projects/%d/comments", pid), map[string]string{"content": "x"}, ""), 401)
	want(t, E.do(t, "POST", fmt.Sprintf("/api/v1/projects/%d/comments", pid), map[string]string{"content": "   "}, tok), 422)
	c := E.do(t, "POST", fmt.Sprintf("/api/v1/projects/%d/comments", pid), map[string]string{"content": "Nice project"}, tok)
	want(t, c, 201)
	cid := num(dig(c.Body, "data", "id"))
	rp := E.do(t, "POST", fmt.Sprintf("/api/v1/comments/%d/reply", cid), map[string]string{"content": "A reply"}, tok)
	want(t, rp, 201)
	if num(dig(rp.Body, "data", "parent_id")) != cid {
		t.Fatal("reply not attached to parent")
	}
	want(t, E.do(t, "POST", "/api/v1/posts/1/comments", map[string]string{"content": "Blog comment"}, tok), 201)
	list := E.do(t, "GET", fmt.Sprintf("/api/v1/projects/%d/comments", pid), nil, "")
	if len(list.Body["data"].([]any)) != 2 {
		t.Fatalf("expected 2 comments, got %v", list.Body["data"])
	}
	want(t, E.do(t, "POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), nil, tok), 200)
	// someone else cannot delete it, admin can hide it
	other, _ := register(t, "other@test.dev")
	want(t, E.do(t, "DELETE", fmt.Sprintf("/api/v1/comments/%d", cid), nil, other), 403)
	at := adminTok(t)
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/comments/%d", cid), map[string]string{"status": "HIDDEN"}, at), 200)
	list = E.do(t, "GET", fmt.Sprintf("/api/v1/projects/%d/comments", pid), nil, "")
	if len(list.Body["data"].([]any)) != 1 {
		t.Fatal("hidden comment still visible")
	}
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/comments/%d", cid), map[string]string{"status": "VISIBLE"}, at), 200)
	want(t, E.do(t, "DELETE", fmt.Sprintf("/api/v1/comments/%d", cid), nil, tok), 204)
	// favorites
	for i := 0; i < 2; i++ {
		want(t, E.do(t, "POST", "/api/v1/products/1/favorite", nil, tok), 200)
	}
	f := E.do(t, "GET", "/api/v1/me/favorites", nil, tok)
	if len(f.Body["data"].([]any)) != 1 {
		t.Fatal("favorite duplicated")
	}
	want(t, E.do(t, "DELETE", "/api/v1/products/1/favorite", nil, tok), 200)
	want(t, E.do(t, "POST", "/api/v1/products/9999/favorite", nil, tok), 404)
}

func TestCartOrderPaymentFlow(t *testing.T) {
	tok, uid := register(t, "buyer@test.dev")
	want(t, E.do(t, "GET", "/api/v1/cart", nil, ""), 401)
	want(t, E.do(t, "POST", "/api/v1/cart/items", map[string]any{"product_id": 9999, "quantity": 1}, tok), 404)
	want(t, E.do(t, "POST", "/api/v1/cart/items", map[string]any{"product_id": 5, "quantity": 500}, tok), 422)
	want(t, E.do(t, "POST", "/api/v1/cart/items", map[string]any{"product_id": 5, "quantity": 1}, tok), 200)
	want(t, E.do(t, "POST", "/api/v1/cart/items", map[string]any{"product_id": 3, "quantity": 2}, tok), 200)
	c := E.do(t, "PATCH", "/api/v1/cart/items/3", map[string]any{"quantity": 1}, tok)
	want(t, c, 200)
	if dig(c.Body, "data", "subtotal").(float64) != 99+59 {
		t.Fatalf("cart subtotal wrong: %v", c.Body["data"])
	}
	// invalid orders
	cust := map[string]string{"name": "Buyer", "email": "buyer@test.dev", "phone": "+998 90 123 45 67", "country": "UZ"}
	want(t, E.do(t, "POST", "/api/v1/orders", map[string]any{"customer": map[string]string{"name": "", "email": "x"}}, tok), 422)
	want(t, E.do(t, "POST", "/api/v1/orders", map[string]any{"customer": cust, "items": []any{map[string]any{"product_id": 9999, "quantity": 1}}}, tok), 422)
	want(t, E.do(t, "POST", "/api/v1/orders", map[string]any{"customer": cust, "items": []any{map[string]any{"product_id": 5, "quantity": 0}}}, tok), 422)
	want(t, E.do(t, "POST", "/api/v1/orders", map[string]any{"customer": cust, "coupon": "NOPE"}, tok), 422)
	// client-sent prices/totals are ignored; idempotency key prevents duplicate orders
	body := map[string]any{"customer": cust, "total": 1, "price": 1, "items": []any{map[string]any{"product_id": 5, "quantity": 1, "price": 0.01}}}
	o1 := E.do(t, "POST", "/api/v1/orders", body, tok, "Idempotency-Key", "key-1")
	want(t, o1, 201)
	if dig(o1.Body, "data", "total").(float64) != 99 {
		t.Fatalf("order trusted client price: %v", o1.Body["data"])
	}
	o2 := E.do(t, "POST", "/api/v1/orders", body, tok, "Idempotency-Key", "key-1")
	want(t, o2, 200)
	if dig(o1.Body, "data", "id") != dig(o2.Body, "data", "id") {
		t.Fatal("duplicate order created for same idempotency key")
	}
	var n int
	E.db.QueryRow(`SELECT count(*) FROM orders WHERE user_id=$1`, uid).Scan(&n)
	if n != 1 {
		t.Fatalf("expected 1 order, got %d", n)
	}
	oid := num(dig(o1.Body, "data", "id"))
	// coupon discount computed server-side
	oc := E.do(t, "POST", "/api/v1/orders", map[string]any{"customer": cust, "coupon": "half", "items": []any{map[string]any{"product_id": 5, "quantity": 1}}}, tok)
	want(t, oc, 201)
	if dig(oc.Body, "data", "total").(float64) != 49.5 || dig(oc.Body, "data", "discount").(float64) != 49.5 {
		t.Fatalf("coupon math wrong: %v", oc.Body["data"])
	}
	// no access before payment
	want(t, E.do(t, "GET", "/api/v1/products/5/access", nil, tok), 403)
	// another user cannot touch the order
	other, _ := register(t, "stranger@test.dev")
	want(t, E.do(t, "GET", fmt.Sprintf("/api/v1/orders/%d", oid), nil, other), 404)
	want(t, E.do(t, "POST", "/api/v1/payments", map[string]any{"order_id": oid}, other), 404)
	// create payment (idempotent while pending)
	p1 := E.do(t, "POST", "/api/v1/payments", map[string]any{"order_id": oid}, tok)
	want(t, p1, 201)
	p2 := E.do(t, "POST", "/api/v1/payments", map[string]any{"order_id": oid}, tok)
	if dig(p1.Body, "data", "payment_id") != dig(p2.Body, "data", "payment_id") {
		t.Fatal("second pending payment created")
	}
	pid := num(dig(p1.Body, "data", "payment_id"))
	// forged webhooks are rejected: bad signature, and unsigned success claims from the browser
	var ref string
	E.db.QueryRow(`SELECT provider_ref FROM payments WHERE id=$1`, pid).Scan(&ref)
	evt := fmt.Sprintf(`{"event_id":"evt_forged","type":"payment.succeeded","payment_ref":"%s","amount_cents":9900,"currency":"USD"}`, ref)
	want(t, E.do(t, "POST", "/api/v1/payments/webhook", evt, "", "X-Signature", "deadbeef"), 401)
	want(t, E.do(t, "POST", "/api/v1/payments/webhook", evt, ""), 401)
	var st string
	E.db.QueryRow(`SELECT status FROM orders WHERE id=$1`, oid).Scan(&st)
	if st != "PENDING" {
		t.Fatal("order changed by forged webhook")
	}
	// correctly signed event with the wrong amount is refused
	tp := E.srv.Pay.(*payments.Test)
	bad := fmt.Sprintf(`{"event_id":"evt_bad_amount","type":"payment.succeeded","payment_ref":"%s","amount_cents":1,"currency":"USD"}`, ref)
	want(t, E.do(t, "POST", "/api/v1/payments/webhook", bad, "", "X-Signature", tp.Sign([]byte(bad))), 422)
	// genuine event: order paid, access granted, duplicate delivery is harmless
	good := fmt.Sprintf(`{"event_id":"evt_good_1","type":"payment.succeeded","payment_ref":"%s","amount_cents":9900,"currency":"USD"}`, ref)
	w1 := E.do(t, "POST", "/api/v1/payments/webhook", good, "", "X-Signature", tp.Sign([]byte(good)))
	want(t, w1, 200)
	w2 := E.do(t, "POST", "/api/v1/payments/webhook", good, "", "X-Signature", tp.Sign([]byte(good)))
	want(t, w2, 200)
	if dig(w2.Body, "data", "duplicate") != true {
		t.Fatal("duplicate webhook not detected")
	}
	var access, events int
	E.db.QueryRow(`SELECT count(*) FROM product_access WHERE user_id=$1 AND status='ACTIVE'`, uid).Scan(&access)
	E.db.QueryRow(`SELECT count(*) FROM payment_events WHERE event_id='evt_good_1'`).Scan(&events)
	if access != 1 || events != 1 {
		t.Fatalf("access=%d events=%d", access, events)
	}
	o := E.do(t, "GET", fmt.Sprintf("/api/v1/orders/%d", oid), nil, tok)
	if dig(o.Body, "data", "status") != "PAID" || dig(o.Body, "data", "payment_status") != "PAID" {
		t.Fatalf("order not paid: %v", o.Body["data"])
	}
	pu := E.do(t, "GET", "/api/v1/me/purchases", nil, tok)
	if len(pu.Body["data"].([]any)) != 1 {
		t.Fatal("purchase missing")
	}
	want(t, E.do(t, "GET", "/api/v1/products/5/access", nil, tok), 200)
	// paid product left the server-side cart, unpaid one remains
	cart := E.do(t, "GET", "/api/v1/cart", nil, tok)
	if len(dig(cart.Body, "data").(map[string]any)["items"].([]any)) != 1 {
		t.Fatalf("cart not converted: %v", cart.Body["data"])
	}
	// paid orders cannot be paid again
	want(t, E.do(t, "POST", "/api/v1/payments", map[string]any{"order_id": oid}, tok), 409)
	// admin changes order status
	at := adminTok(t)
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/orders/%d", oid), map[string]string{"status": "BOGUS"}, at), 422)
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/orders/%d", oid), map[string]string{"status": "COMPLETED"}, at), 200)
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/orders/%d", oid), map[string]string{"status": "REFUNDED"}, at), 200)
	want(t, E.do(t, "GET", "/api/v1/products/5/access", nil, tok), 403) // access revoked on refund
	// test-provider simulation goes through the same verified path
	o3 := E.do(t, "POST", "/api/v1/orders", map[string]any{"customer": cust, "items": []any{map[string]any{"product_id": 3, "quantity": 1}}}, tok)
	p3 := E.do(t, "POST", "/api/v1/payments", map[string]any{"order_id": num(dig(o3.Body, "data", "id"))}, tok)
	want(t, E.do(t, "POST", "/api/v1/payments/test/complete", map[string]any{"payment_id": num(dig(p3.Body, "data", "payment_id")), "outcome": "success"}, tok), 200)
	o3b := E.do(t, "GET", fmt.Sprintf("/api/v1/orders/%d", num(dig(o3.Body, "data", "id"))), nil, tok)
	if dig(o3b.Body, "data", "status") != "PAID" {
		t.Fatal("test provider flow did not pay the order")
	}
	// audit log written for the admin action
	var logs int
	E.db.QueryRow(`SELECT count(*) FROM audit_logs WHERE action='order status changed'`).Scan(&logs)
	if logs < 2 {
		t.Fatal("audit log missing")
	}
}

func TestAdminContentCRUD(t *testing.T) {
	at := adminTok(t)
	want(t, E.do(t, "POST", "/api/v1/admin/products", map[string]any{"title": "", "category": "nope", "price": -5}, at), 422)
	p := E.do(t, "POST", "/api/v1/admin/products", map[string]any{"title": "New Bot", "category": "bots", "price": 19.99, "description": "d", "technologies": []string{"Go", "PostgreSQL"}, "features": []string{"A", "B"}, "download_url": "private/x.zip", "published": false}, at)
	want(t, p, 201)
	id := num(dig(p.Body, "data", "id"))
	want(t, E.do(t, "GET", "/api/v1/products/"+str(dig(p.Body, "data", "slug")), nil, ""), 404) // drafts are not public
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/products/%d", id), map[string]any{"published": true, "featured": true, "price": 24.5}, at), 200)
	pub := E.do(t, "GET", "/api/v1/products/"+str(dig(p.Body, "data", "slug")), nil, "")
	want(t, pub, 200)
	if _, leaked := pub.Body["data"].(map[string]any)["download_url"]; leaked || len(pub.Body["data"].(map[string]any)["technologies"].([]any)) != 2 {
		t.Fatalf("public product wrong: %v", pub.Body["data"])
	}
	want(t, E.do(t, "DELETE", fmt.Sprintf("/api/v1/admin/products/%d", id), nil, at), 204)
	want(t, E.do(t, "DELETE", fmt.Sprintf("/api/v1/admin/products/%d", id), nil, at), 404)
	// blog
	po := E.do(t, "POST", "/api/v1/admin/posts", map[string]any{"title": "Hello World Post", "category": "AI", "content": strings.Repeat("word ", 450), "published": true, "tags": []string{"go"}}, at)
	want(t, po, 201)
	if dig(po.Body, "data", "reading_time") != "3 min" || dig(po.Body, "data", "published_at") == nil {
		t.Fatalf("post fields wrong: %v", po.Body["data"])
	}
	want(t, E.do(t, "POST", "/api/v1/admin/posts", map[string]any{"title": "x", "category": "Nope"}, at), 422)
	want(t, E.do(t, "GET", "/api/v1/posts/hello-world-post", nil, ""), 200)
	// project
	pj := E.do(t, "POST", "/api/v1/admin/projects", map[string]any{"title": "Fresh Project", "category": "SaaS", "features": []string{"f1"}, "technologies": []string{"Go"}, "published": true}, at)
	want(t, pj, 201)
	want(t, E.do(t, "PATCH", fmt.Sprintf("/api/v1/admin/projects/%d", num(dig(pj.Body, "data", "id"))), map[string]any{"published": false}, at), 200)
	want(t, E.do(t, "GET", "/api/v1/projects/fresh-project", nil, ""), 404)
	// portfolio content
	want(t, E.do(t, "PUT", "/api/v1/admin/portfolio", map[string]any{"skills": []any{map[string]any{"category": "x", "name": "Y", "level": 500}}}, at), 422)
	want(t, E.do(t, "PUT", "/api/v1/admin/portfolio", map[string]any{"hero_title": "Hi [[there]]", "skills": []any{map[string]any{"category": "backend", "name": "Go", "level": 90}}}, at), 200)
	pf := E.do(t, "GET", "/api/v1/portfolio", nil, "")
	if dig(pf.Body, "data", "hero_title") != "Hi [[there]]" || len(dig(pf.Body, "data").(map[string]any)["skills"].([]any)) != 1 {
		t.Fatalf("portfolio not updated: %v", pf.Body["data"])
	}
	// contact
	want(t, E.do(t, "POST", "/api/v1/contact", map[string]any{"name": "A", "email": "bad", "message": "hi"}, ""), 422)
	want(t, E.do(t, "POST", "/api/v1/contact", map[string]any{"name": "A", "email": "a@b.co", "subject": "Hi", "message": "Hello there"}, ""), 201)
	cm := E.do(t, "GET", "/api/v1/admin/contact-messages?status=new", nil, at)
	want(t, cm, 200)
	if num(dig(cm.Body, "pagination", "total")) != 1 {
		t.Fatal("contact message not stored")
	}
	d := E.do(t, "GET", "/api/v1/admin/dashboard", nil, at)
	if num(dig(d.Body, "data", "unread_messages")) != 1 || len(dig(d.Body, "data").(map[string]any)["revenue_7d"].([]any)) != 7 {
		t.Fatalf("dashboard wrong: %v", d.Body["data"])
	}
}

func TestMediaUploadAndProtectedFiles(t *testing.T) {
	at := adminTok(t)
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0x1f, 0x15, 0xc4, 0x89}
	up := func(name string, data []byte, priv bool, tok string) res {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, name))
		fw, _ := mw.CreatePart(h)
		fw.Write(data)
		if priv {
			mw.WriteField("private", "true")
		}
		mw.Close()
		req, _ := http.NewRequest("POST", E.ts.URL+"/api/v1/admin/media", &b)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var m map[string]any
		json.NewDecoder(resp.Body).Decode(&m)
		return res{Code: resp.StatusCode, Body: m}
	}
	want(t, up("evil.html", []byte("<script>alert(1)</script>"), false, at), 415)
	user, _ := register(t, "uploader@test.dev")
	want(t, up("a.png", png, false, user), 403)
	r := up("a.png", png, false, at)
	want(t, r, 201)
	key := str(dig(r.Body, "data", "storage_key"))
	g, _ := http.Get(E.ts.URL + "/files/" + key)
	if g.StatusCode != 200 {
		t.Fatalf("public file not served: %d", g.StatusCode)
	}
	zip := []byte("PK\x03\x04\x14\x00\x00\x00\x08\x00padding-padding-padding")
	pr := up("secret.zip", zip, true, at)
	want(t, pr, 201)
	pk := str(dig(pr.Body, "data", "storage_key"))
	g, _ = http.Get(E.ts.URL + "/files/" + pk)
	if g.StatusCode != 403 {
		t.Fatalf("private file served without signature: %d", g.StatusCode)
	}
	signed, _ := E.srv.Store.GetURL(pk, time.Minute)
	g, _ = http.Get(strings.Replace(signed, "http://localhost", E.ts.URL, 1))
	if g.StatusCode != 200 {
		t.Fatalf("signed url rejected: %d", g.StatusCode)
	}
	g, _ = http.Get(E.ts.URL + "/files/../../etc/passwd")
	if g.StatusCode == 200 {
		t.Fatal("path traversal")
	}
}

func TestRateLimitAndMalformedInput(t *testing.T) {
	e := newServer(E.db, func(c *config.Config) { c.AuthRate, c.AuthBurst = 1, 3 })
	defer e.ts.Close()
	got429 := false
	for i := 0; i < 8; i++ {
		r := e.do(t, "POST", "/api/v1/auth/login", map[string]string{"email": "x@y.zz", "password": "nope"}, "")
		if r.Code == 429 {
			got429 = true
		}
	}
	if !got429 {
		t.Fatal("rate limit did not trigger")
	}
	want(t, E.do(t, "POST", "/api/v1/auth/login", "{not json", ""), 400)
	want(t, E.do(t, "GET", "/api/v1/products/", nil, ""), 404)
}
