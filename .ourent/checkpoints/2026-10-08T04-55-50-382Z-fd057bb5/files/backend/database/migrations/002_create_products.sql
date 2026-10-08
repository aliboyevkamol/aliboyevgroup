CREATE TABLE products (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
  slug TEXT NOT NULL UNIQUE,
  category TEXT NOT NULL CHECK (category IN ('websites','saas','bots','ai','templates','automation')),
  description TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL DEFAULT '',
  price NUMERIC(12,2) NOT NULL CHECK (price >= 0),
  old_price NUMERIC(12,2) CHECK (old_price IS NULL OR old_price >= 0),
  currency TEXT NOT NULL DEFAULT 'USD',
  cover_image TEXT NOT NULL DEFAULT '',
  demo_url TEXT NOT NULL DEFAULT '',
  demo_type TEXT NOT NULL DEFAULT '' CHECK (demo_type IN ('','iframe','external','internal')),
  live_url TEXT NOT NULL DEFAULT '',
  download_url TEXT NOT NULL DEFAULT '',
  mock_variant INT NOT NULL DEFAULT 1 CHECK (mock_variant BETWEEN 1 AND 5),
  featured BOOLEAN NOT NULL DEFAULT false,
  published BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX products_list_idx ON products (published, category, created_at DESC);
CREATE TABLE product_images (
  id BIGSERIAL PRIMARY KEY,
  product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  url TEXT NOT NULL, alt TEXT NOT NULL DEFAULT '', position INT NOT NULL DEFAULT 0
);
CREATE TABLE product_features (
  id BIGSERIAL PRIMARY KEY,
  product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  text TEXT NOT NULL, position INT NOT NULL DEFAULT 0
);
CREATE TABLE product_technologies (
  id BIGSERIAL PRIMARY KEY,
  product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  name TEXT NOT NULL, position INT NOT NULL DEFAULT 0,
  UNIQUE (product_id, name)
);
CREATE INDEX product_tech_name_idx ON product_technologies (lower(name));
