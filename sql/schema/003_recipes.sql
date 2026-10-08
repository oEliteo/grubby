-- +goose Up
CREATE TABLE recipes(
   id UUID PRIMARY KEY,
   user_id UUID REFERENCES users(id) ON DELETE SET NULL,
   created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   title TEXT NOT NULL,
   description TEXT,
   is_public BOOLEAN NOT NULL DEFAULT FALSE,
   ingredients JSONB NOT NULL,
   nutrition JSONB DEFAULT NULL,
   instructions TEXT NOT NULL,
   nutrition_status TEXT NOT NULL DEFAULT 'unprocessed',
   deleted_at TIMESTAMPTZ DEFAULT NULL
);

-- +goose Down
DROP TABLE IF EXISTS recipes;
