CREATE TABLE tokens (
  id         BIGSERIAL   NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  code       TEXT        NOT NULL,
  access     TEXT        NOT NULL,
  refresh    TEXT        NOT NULL,
  data       JSONB       NOT NULL,
  CONSTRAINT tokens_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_tokens_expires_at ON tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_tokens_code ON tokens (code);
CREATE INDEX IF NOT EXISTS idx_tokens_access ON tokens (access);
CREATE INDEX IF NOT EXISTS idx_tokens_refresh ON tokens (refresh);
