-- +goose Up
CREATE TABLE recipes(
   id UUID PRIMARY KEY,
   user_id UUID REFERENCES users(id) ON DELETE SET NULL,
   created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   title VARCHAR(150) NOT NULL CHECK (length(trim(title)) > 0),
   description VARCHAR(1000),
   is_public BOOLEAN NOT NULL DEFAULT FALSE,
   ingredients JSONB NOT NULL CHECK(
      jsonb_typeof(ingredients) = 'array' AND
      jsonb_array_length(ingredients) > 0 AND
      jsonb_array_length(ingredients) <= 50
   ),
   nutrition JSONB DEFAULT NULL,
   instructions JSONB NOT NULL CHECK (
      jsonb_typeof(instructions) = 'array' AND
      jsonb_array_length(instructions) > 0 AND
      jsonb_array_length(instructions) <= 50
   ),
   nutrition_status TEXT NOT NULL DEFAULT 'unprocessed',
   deleted_at TIMESTAMPTZ DEFAULT NULL
);

-- +goose Down
DROP TABLE IF EXISTS recipes;
