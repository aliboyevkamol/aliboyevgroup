# Portfolio backend (Go + PostgreSQL)

REST API for the portfolio website: store, blog, projects, comments, likes, favorites, cart, orders,
payments (provider abstraction + webhook), protected downloads, contact form, admin panel API.
The frontend (HTML/CSS/vanilla JS) lives one folder up and talks to this API.

* Go 1.24, standard `net/http` router, `database/sql` + `lib/pq` (the only third-party dependency)
* PostgreSQL 16, 15 SQL migrations (embedded, applied automatically), seed content migrated from the old frontend
* Auth: short-lived JWT access token (memory only) + rotating refresh token in an **HttpOnly** cookie
* Passwords: PBKDF2-HMAC-SHA256, 600 000 iterations (standard library). Swap in Argon2id by replacing `internal/auth/password.go`
* OpenAPI 3 spec at `docs/openapi.yaml`, served at `/api/v1/openapi.yaml`, Swagger UI at `/docs`

## Layout

```
backend/
  cmd/server/main.go        entry point: serve | migrate | seed | create-admin
  config/                   environment configuration + production safety checks
  database/                 migrations/*.sql (001..015), seeds/content.json, migration + seed runner
  middleware/               request id, logging, panic recovery, security headers, CORS, rate limiting
  internal/auth/            password hashing, JWT
  internal/payments/        Provider interface + development "test" provider
  internal/api/             routes and handlers (auth, catalog, social, commerce, admin, misc) + tests
  pkg/                      response envelope, logger, storage (local driver), mail, slug
  docs/                     openapi.yaml
  Dockerfile  docker-compose.yml  .env.example
```

## Run locally

### With Docker (recommended)
```bash
cd backend
cp .env.example .env            # change the secrets for anything but local testing
docker compose up -d --build
docker compose exec backend /app/server create-admin -email you@example.com -password 'a-long-password-12+'
```
The backend also serves the website (`STATIC_DIR=/frontend`), so open **http://localhost:8080**.
Admin panel: **http://localhost:8080/admin/**. API docs: http://localhost:8080/docs.

### Without Docker
Needs Go 1.24+ and a PostgreSQL database.
```bash
cd backend
export DATABASE_URL='postgres://user:pass@localhost:5432/kamol?sslmode=disable'
export STATIC_DIR=..                      # serve the website from this API (single origin)
go run ./cmd/server                       # migrates + seeds automatically (AUTO_MIGRATE / AUTO_SEED)
go run ./cmd/server create-admin -email you@example.com -password 'a-long-password-12+'
```
Other commands: `go run ./cmd/server migrate`, `go run ./cmd/server seed`.

### Open it on a phone
Run the backend on your computer and open `http://<computer-LAN-IP>:8080` on the phone (same Wi-Fi).
The frontend uses the same origin automatically. If you host the frontend elsewhere, set `FRONTEND_URL` (CORS)
and, in the browser console of that site, `localStorage.kamol_api_origin = 'https://api.example.com'`.

### First admin
`create-admin` (above) creates or promotes an admin. Alternatively set `ADMIN_EMAIL` + `ADMIN_PASSWORD` once; an admin
is created at start-up only if none exists. Remove them afterwards. There are no default credentials.

## Configuration

See `.env.example`. In `APP_ENV=production` the server refuses to start unless `JWT_SECRET`, `JWT_REFRESH_SECRET`
and `PAYMENT_WEBHOOK_SECRET` are random values of 32+ characters, and `PAYMENT_PROVIDER=test` is rejected.
Refresh cookies get the `Secure` flag in production, so serve the site over HTTPS.

## Tests
```bash
export TEST_DATABASE_URL='postgres://user:pass@localhost:5432/kamol_test?sslmode=disable'   # DATABASE IS WIPED
go test ./...
```
The suite resets the schema, migrates, seeds and exercises: envelope and security headers, registration validation,
duplicate email, wrong password, refresh rotation + reuse detection + foreign-origin refusal, blocked users,
admin-only endpoints (401/403), product listing/filter/search/sort/pagination, public data never leaking `download_url`,
duplicate likes and favorites, comments/replies/moderation, cart math, invalid orders, client prices ignored,
idempotent orders, coupons, forged / unsigned / wrong-amount / duplicate webhooks, product access grant and revoke,
admin CRUD, portfolio content, contact, media upload type checks, signed private downloads, path traversal, rate limiting.

## Payments

```
POST /orders  ->  POST /payments  ->  provider  ->  POST /payments/webhook  ->  order PAID + product_access
```
* Totals are computed in the database layer from current product prices. Client prices/totals are ignored.
* Only a webhook with a valid signature, a known `payment_ref` and a matching amount can mark an order paid.
  Events are idempotent (`payment_events.event_id` primary key) and applied in one transaction
  (payment, order, access, cart cleanup).
* Card data never reaches the backend. The frontend card form is validated in the browser only; a real provider must use its hosted / tokenized flow.
* The built-in `test` provider moves no money. In development `POST /payments/test/complete` simulates the provider page by
  signing an event and sending it through the same webhook code path.
* **To go live** implement `payments.Provider` (`CreatePayment`, `VerifyWebhook`, `GetStatus`, `Refund`) for your provider
  (Stripe, Payme, Click, ...), register it in `payments.New`, set `PAYMENT_PROVIDER` and its secrets, and return the provider's
  hosted `checkout_url` from `CreatePayment` (the frontend redirects to it automatically).

## Files and downloads
Uploads go through `storage.Storage`. The local driver is built in; for S3 / Cloudflare R2 / Spaces add an adapter that
implements `Upload`, `Delete`, `GetURL`. Keys under `public/` are public images; everything else is private and only
reachable through short-lived signed URLs after `GET /products/{id}/access` (purchasers and admins only).
A product's `download_url` may be a private storage key or an external https URL.

## Deployment (VPS / Docker server)
1. Install Docker, copy the project, `cd backend && cp .env.example .env` and set: `APP_ENV=production`, three random secrets
   (`openssl rand -hex 32`), `POSTGRES_PASSWORD`, `PUBLIC_URL=https://your-domain`, `FRONTEND_URL=https://your-domain`,
   a real `PAYMENT_PROVIDER`, SMTP settings, `TRUST_PROXY=true` (when behind nginx/Caddy).
2. Put a TLS reverse proxy (Caddy or nginx) in front of port 8080 and close port 5432 to the internet.
3. `docker compose up -d --build`, then `create-admin`. Migrations run at start-up (or `server migrate`).
4. Upgrades: pull, `docker compose up -d --build`. Migrations are forward-only; add new files as `0NN_name.sql`.

### Backups
```bash
docker compose exec -T postgres pg_dump -U kamol -Fc kamol > backup-$(date +%F).dump        # database
docker run --rm -v backend_uploads:/data -v "$PWD":/out alpine tar czf /out/uploads-$(date +%F).tgz -C /data .   # files
docker compose exec -T postgres pg_restore -U kamol -d kamol --clean < backup.dump           # restore
```
Run the dump from cron and copy it off the server.

## Production security checklist
- [ ] `APP_ENV=production`, strong unique secrets, HTTPS everywhere, HSTS (sent automatically in production)
- [ ] Real payment provider with webhook signature secret; `PAYMENT_PROVIDER=test` removed
- [ ] PostgreSQL not exposed publicly, strong password, regular tested backups
- [ ] `TRUST_PROXY=true` only behind a proxy that sets `X-Forwarded-For` (otherwise rate limits can be bypassed)
- [ ] Rate limits are in-memory per instance; use the proxy or a shared limiter if you run several instances
- [ ] `STATIC_DIR` serves only whitelisted file types and never `backend/` or dot-files; prefer serving static files from the proxy
- [ ] Admin accounts have long passwords; consider adding 2FA and an IP allow-list for `/admin/`
- [ ] Add a Content-Security-Policy for the static site at the proxy (the API already sends a strict one)
- [ ] Monitor logs (structured JSON in production; passwords, tokens and card data are never logged)
- [ ] Review `audit_logs` periodically (admin actions and payment events are recorded)

## Known limits
- Password hashing is PBKDF2 (standard library), not Argon2id.
- Email (password reset) only logs a message unless SMTP is configured.
- The `roles` table from the spec is a `users.role` column (USER, ADMIN, EDITOR, MODERATOR); only USER and ADMIN have behavior today.
- `comment_likes` is folded into the single `likes` table (UNIQUE(user_id, target_type, target_id)).
- About / services / experience content is stored and editable through `PUT /admin/portfolio`, but the current public pages
  still render those sections from static HTML (hero text, social links and skills are live from the API).
