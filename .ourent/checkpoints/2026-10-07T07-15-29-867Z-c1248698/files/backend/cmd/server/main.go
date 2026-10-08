package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"kamolportfolio/backend/config"
	"kamolportfolio/backend/database"
	"kamolportfolio/backend/internal/api"
	"kamolportfolio/backend/internal/auth"
	"kamolportfolio/backend/internal/payments"
	"kamolportfolio/backend/pkg/logger"
	"kamolportfolio/backend/pkg/mail"
	"kamolportfolio/backend/pkg/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	log := logger.New(cfg.Env)
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Error("database open failed", "err", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx := context.Background()
	for i := 0; ; i++ { // wait for the database (docker compose start-up)
		if err = db.PingContext(ctx); err == nil {
			break
		}
		if i > 30 {
			log.Error("database unreachable", "err", err)
			os.Exit(1)
		}
		time.Sleep(time.Second)
	}

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "migrate":
		fatal(log, database.Migrate(ctx, db, log))
		return
	case "seed":
		fatal(log, database.Migrate(ctx, db, log))
		fatal(log, database.Seed(ctx, db, log))
		return
	case "create-admin":
		fs := flag.NewFlagSet("create-admin", flag.ExitOnError)
		email, pw, name := fs.String("email", "", "admin email"), fs.String("password", "", "admin password (min 12 chars)"), fs.String("name", "Admin", "display name")
		_ = fs.Parse(os.Args[2:])
		fatal(log, database.Migrate(ctx, db, log))
		fatal(log, createAdmin(ctx, db, *name, *email, *pw))
		fmt.Println("admin ready:", *email)
		return
	case "serve":
	default:
		fmt.Fprintln(os.Stderr, "usage: server [serve|migrate|seed|create-admin -email e -password p]")
		os.Exit(2)
	}

	if cfg.AutoMigrate {
		fatal(log, database.Migrate(ctx, db, log))
	}
	if cfg.AutoSeed {
		fatal(log, database.Seed(ctx, db, log))
	}
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		var n int
		_ = db.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE role='ADMIN'`).Scan(&n)
		if n == 0 {
			fatal(log, createAdmin(ctx, db, "Admin", cfg.AdminEmail, cfg.AdminPassword))
			log.Info("initial admin created", "email", cfg.AdminEmail)
		}
	}
	prov, err := payments.New(cfg.PaymentProvider, cfg.WebhookSecret)
	fatal(log, err)
	st, err := storage.New(cfg.StorageDriver, cfg.StoragePath, cfg.PublicURL, cfg.JWTSecret)
	fatal(log, err)
	var mailer mail.Mailer = mail.Log{L: log, ShowBody: !cfg.IsProd()}
	if cfg.SMTPHost != "" {
		mailer = mail.SMTP{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Pass: cfg.SMTPPassword, From: cfg.MailFrom}
	}
	srv := &api.Server{DB: db, Cfg: cfg, Log: log, Pay: prov, Store: st, Mail: mailer}
	hs := &http.Server{Addr: ":" + cfg.Port, Handler: srv.Routes(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
	go func() {
		log.Info("server listening", "port", cfg.Port, "env", cfg.Env, "payment_provider", prov.Name())
		if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	c, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_ = hs.Shutdown(c)
}

func fatal(log interface{ Error(string, ...any) }, err error) {
	if err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func createAdmin(ctx context.Context, db *sql.DB, name, email, pw string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(pw) < 12 {
		return errors.New("email is required and password must be at least 12 characters")
	}
	h, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO users (name,email,password_hash,role) VALUES ($1,$2,$3,'ADMIN')
ON CONFLICT ((lower(email))) DO UPDATE SET role='ADMIN', password_hash=EXCLUDED.password_hash, status='ACTIVE', updated_at=now()`, name, email, h)
	return err
}
