CREATE TABLE post_categories (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
INSERT INTO post_categories (name) VALUES ('Development'),('AI'),('SaaS'),('Automation'),('Business'),('Tutorials');
CREATE TABLE posts (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 250),
  slug TEXT NOT NULL UNIQUE,
  excerpt TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL DEFAULT '',
  cover_image TEXT NOT NULL DEFAULT '',
  category_id INT NOT NULL REFERENCES post_categories(id),
  reading_time TEXT NOT NULL DEFAULT '',
  author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  author_name TEXT NOT NULL DEFAULT '',
  published BOOLEAN NOT NULL DEFAULT false,
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX posts_list_idx ON posts (published, category_id, published_at DESC);
CREATE TABLE post_tags (
  post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  tag TEXT NOT NULL,
  PRIMARY KEY (post_id, tag)
);
