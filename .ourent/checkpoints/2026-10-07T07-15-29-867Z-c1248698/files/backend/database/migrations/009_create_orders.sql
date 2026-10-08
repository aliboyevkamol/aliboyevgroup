CREATE TABLE coupons (
  code TEXT PRIMARY KEY,
  percent INT NOT NULL CHECK (percent BETWEEN 1 AND 100),
  active BOOLEAN NOT NULL DEFAULT true,
  expires_at TIMESTAMPTZ
);
CREATE TABLE orders (
  id BIGSERIAL PRIMARY KEY,
  order_number TEXT NOT NULL UNIQUE,
  user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  customer_name TEXT NOT NULL,
  customer_email TEXT NOT NULL,
  customer_phone TEXT NOT NULL DEFAULT '',
  country TEXT NOT NULL DEFAULT '',
  subtotal NUMERIC(12,2) NOT NULL CHECK (subtotal >= 0),
  discount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (discount >= 0),
  total NUMERIC(12,2) NOT NULL CHECK (total >= 0),
  currency TEXT NOT NULL DEFAULT 'USD',
  coupon_code TEXT,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','PROCESSING','PAID','COMPLETED','CANCELLED','REFUNDED')),
  payment_status TEXT NOT NULL DEFAULT 'UNPAID' CHECK (payment_status IN ('UNPAID','PENDING','PAID','FAILED','REFUNDED')),
  idempotency_key TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX orders_idem_idx ON orders (user_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX orders_user_idx ON orders (user_id, created_at DESC);
CREATE TABLE order_items (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id BIGINT REFERENCES products(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
  quantity INT NOT NULL CHECK (quantity BETWEEN 1 AND 99)
);
CREATE INDEX order_items_order_idx ON order_items (order_id);
