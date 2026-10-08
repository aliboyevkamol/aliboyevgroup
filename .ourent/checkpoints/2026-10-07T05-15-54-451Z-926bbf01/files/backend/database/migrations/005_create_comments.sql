CREATE TABLE comments (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_id BIGINT REFERENCES posts(id) ON DELETE CASCADE,
  project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE,
  parent_id BIGINT REFERENCES comments(id) ON DELETE CASCADE,
  content TEXT NOT NULL CHECK (length(content) BETWEEN 1 AND 2000),
  status TEXT NOT NULL DEFAULT 'VISIBLE' CHECK (status IN ('VISIBLE','HIDDEN','DELETED','PENDING')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT comments_one_target CHECK ((post_id IS NULL) <> (project_id IS NULL))
);
CREATE INDEX comments_post_idx ON comments (post_id, status, created_at);
CREATE INDEX comments_project_idx ON comments (project_id, status, created_at);
