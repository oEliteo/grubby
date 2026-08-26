-- +goose Up
CREATE TABLE users(
   id UUID PRIMARY KEY,
   created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   email TEXT UNIQUE NOT NULL,
   display_name TEXT UNIQUE NOT NULL,
   hashed_password TEXT NOT NULL,
   is_premium BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down
DROP TABLE IF EXISTS users;
