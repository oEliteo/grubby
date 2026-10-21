-- +goose Up
CREATE TABLE users(
   id UUID PRIMARY KEY,
   created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   email VARCHAR(255) UNIQUE NOT NULL,
   display_name VARCHAR(50) UNIQUE NOT NULL CHECK(length(trim(display_name)) >= 3),
   hashed_password TEXT NOT NULL,
   is_premium BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down
DROP TABLE IF EXISTS users;
