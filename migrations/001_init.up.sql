BEGIN;

CREATE TABLE users (
  id         BIGINT PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE orders (
  id         UUID PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id),
  status     TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_items (
  order_id UUID NOT NULL REFERENCES orders(id),
  sku      TEXT NOT NULL,
  quantity INT NOT NULL,
  PRIMARY KEY (order_id, sku)
);

CREATE TABLE waves (
  id          UUID PRIMARY KEY,
  status      TEXT NOT NULL,
  max_orders  INT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  closed_at   TIMESTAMPTZ
);

CREATE TABLE wave_orders (
  wave_id  UUID NOT NULL REFERENCES waves(id),
  order_id UUID NOT NULL REFERENCES orders(id),
  PRIMARY KEY (wave_id, order_id)
);

CREATE TABLE stocks (
  sku             TEXT PRIMARY KEY,
  total_quantity  INT NOT NULL
);

CREATE TABLE reservations (
  id             SERIAL PRIMARY KEY,
  order_id       UUID NOT NULL REFERENCES orders(id),
  sku            TEXT NOT NULL,
  reserved_qty   INT NOT NULL,
  backorder_qty  INT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE outbox (
  id           SERIAL PRIMARY KEY,
  event_id     UUID NOT NULL,
  event_type   TEXT NOT NULL,
  order_id     UUID NOT NULL,
  user_id      BIGINT NOT NULL,
  status       TEXT NOT NULL,
  occurred_at  TIMESTAMPTZ NOT NULL,
  processed_at TIMESTAMPTZ,
  error        TEXT,
  retry_count  INT NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
