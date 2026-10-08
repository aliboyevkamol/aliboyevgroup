CREATE TABLE projects (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
  slug TEXT NOT NULL UNIQUE,
  category TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  full_description TEXT NOT NULL DEFAULT '',
  problem TEXT NOT NULL DEFAULT '',
  solution TEXT NOT NULL DEFAULT '',
  outcome TEXT NOT NULL DEFAULT '',
  demo_url TEXT NOT NULL DEFAULT '',
  live_url TEXT NOT NULL DEFAULT '',
  github_url TEXT NOT NULL DEFAULT '',
  cover_image TEXT NOT NULL DEFAULT '',
  mock_variant INT NOT NULL DEFAULT 1 CHECK (mock_variant BETWEEN 1 AND 5),
  featured BOOLEAN NOT NULL DEFAULT false,
  published BOOLEAN NOT NULL DEFAULT false,
  position INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE project_images (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  url TEXT NOT NULL, alt TEXT NOT NULL DEFAULT '', position INT NOT NULL DEFAULT 0
);
CREATE TABLE project_features (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  text TEXT NOT NULL, position INT NOT NULL DEFAULT 0
);
CREATE TABLE project_technologies (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL, position INT NOT NULL DEFAULT 0,
  UNIQUE (project_id, name)
);
CREATE TABLE project_architecture (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  step TEXT NOT NULL, position INT NOT NULL DEFAULT 0
);
