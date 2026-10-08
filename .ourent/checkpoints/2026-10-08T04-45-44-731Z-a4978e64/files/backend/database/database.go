// Package database runs migrations and seeds the initial content.
package database

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"kamolportfolio/backend/pkg/slug"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed seeds/content.json
var seedJSON []byte

func Migrate(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(727274)`); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(727274)`)
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	files, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := []string{}
	for _, f := range files {
		names = append(names, f.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		var done bool
		if err := conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, n).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		sqlText, _ := migrationsFS.ReadFile("migrations/" + n)
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(sqlText)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", n, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, n); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Info("migration applied", "version", n)
	}
	return nil
}

type seedFile struct {
	Products []struct {
		Title, Category, Description string
		Price                        float64
		Technologies, Features       []string
		Featured                     bool
		MockVariant                  int `json:"mock_variant"`
	}
	Projects []struct {
		Title, Slug, Category, Description, Problem, Solution, Outcome string
		Features, Technologies, Architecture                           []string
		MockVariant                                                    int `json:"mock_variant"`
	}
	Posts []struct {
		Title, Category, Excerpt, Content, Author string
		ReadingTime                               string `json:"reading_time"`
		PublishedAt                               string `json:"published_at"`
	}
	Skills []struct {
		Category, Name string
		Level          int
	}
	Site struct {
		HeroTitle       string            `json:"hero_title"`
		HeroDescription string            `json:"hero_description"`
		SocialLinks     map[string]string `json:"social_links"`
	}
	Services, Experiences []struct{ Title, Description string }
}

func list(ctx context.Context, tx *sql.Tx, table, fk, col string, id int64, vals []string) error {
	for i, v := range vals {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (%s,%s,position) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, table, fk, col), id, v, i); err != nil {
			return err
		}
	}
	return nil
}

// Seed loads the portfolio content that previously lived in the frontend (only into empty tables).
func Seed(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	var s seedFile
	if err := json.Unmarshal(seedJSON, &s); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var n int
	_ = tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM products)+(SELECT count(*) FROM projects)+(SELECT count(*) FROM posts)`).Scan(&n)
	if n > 0 {
		return nil // already seeded or managed through the admin panel
	}
	for _, p := range s.Products {
		var id int64
		if err := tx.QueryRowContext(ctx, `INSERT INTO products (title,slug,category,description,price,featured,published,mock_variant) VALUES ($1,$2,$3,$4,$5,$6,true,$7) RETURNING id`,
			p.Title, slug.Make(p.Title), p.Category, p.Description, p.Price, p.Featured, p.MockVariant).Scan(&id); err != nil {
			return fmt.Errorf("seed product: %w", err)
		}
		if err := list(ctx, tx, "product_technologies", "product_id", "name", id, p.Technologies); err != nil {
			return err
		}
		if err := list(ctx, tx, "product_features", "product_id", "text", id, p.Features); err != nil {
			return err
		}
	}
	for i, p := range s.Projects {
		var id int64
		if err := tx.QueryRowContext(ctx, `INSERT INTO projects (title,slug,category,description,problem,solution,outcome,mock_variant,featured,published,position) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10) RETURNING id`,
			p.Title, p.Slug, p.Category, p.Description, p.Problem, p.Solution, p.Outcome, p.MockVariant, i < 3, i).Scan(&id); err != nil {
			return fmt.Errorf("seed project: %w", err)
		}
		for _, t := range []struct {
			tbl, col string
			v        []string
		}{{"project_technologies", "name", p.Technologies}, {"project_features", "text", p.Features}, {"project_architecture", "step", p.Architecture}} {
			if err := list(ctx, tx, t.tbl, "project_id", t.col, id, t.v); err != nil {
				return err
			}
		}
	}
	for _, p := range s.Posts {
		var id int64
		if err := tx.QueryRowContext(ctx, `INSERT INTO posts (title,slug,excerpt,content,category_id,reading_time,author_name,published,published_at) VALUES ($1,$2,$3,$4,(SELECT id FROM post_categories WHERE name=$5),$6,$7,true,$8) RETURNING id`,
			p.Title, slug.Make(p.Title), p.Excerpt, p.Content, p.Category, p.ReadingTime, p.Author, p.PublishedAt).Scan(&id); err != nil {
			return fmt.Errorf("seed post: %w", err)
		}
	}
	for i, k := range s.Skills {
		if _, err := tx.ExecContext(ctx, `INSERT INTO skills (category,name,level,position) VALUES ($1,$2,$3,$4)`, k.Category, k.Name, k.Level, i); err != nil {
			return err
		}
	}
	for i, k := range s.Services {
		tx.ExecContext(ctx, `INSERT INTO services (title,description,position) VALUES ($1,$2,$3)`, k.Title, k.Description, i)
	}
	for i, k := range s.Experiences {
		tx.ExecContext(ctx, `INSERT INTO experiences (title,description,position) VALUES ($1,$2,$3)`, k.Title, k.Description, i)
	}
	links, _ := json.Marshal(s.Site.SocialLinks)
	for k, v := range map[string]any{"hero_title": s.Site.HeroTitle, "hero_description": s.Site.HeroDescription, "social_links": json.RawMessage(links)} {
		b, _ := json.Marshal(v)
		if _, err := tx.ExecContext(ctx, `INSERT INTO site_settings (key,value) VALUES ($1,$2) ON CONFLICT (key) DO NOTHING`, k, string(b)); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Info("seed content loaded", "products", len(s.Products), "projects", len(s.Projects), "posts", len(s.Posts))
	return nil
}
