CREATE TABLE media (
  id BIGSERIAL PRIMARY KEY,
  uploader_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  storage_key TEXT NOT NULL UNIQUE,
  url TEXT NOT NULL,
  filename TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  is_private BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
