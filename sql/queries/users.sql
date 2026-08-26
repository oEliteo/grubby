-- name: CreateUser :one
INSERT INTO users (
   id,
   created_at,
   updated_at,
   email,
   display_name,
   hashed_password,
   is_premium
) VALUES (
   $1, NOW(), NOW(), $2, $3, $4, $5
)
RETURNING id, created_at, updated_at, email, display_name, is_premium;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, created_at, updated_at, email, display_name, is_premium
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET display_name = $1,
   email = $2,
   updated_at = NOW()
WHERE id = $3
RETURNING id, created_at, updated_at, display_name, is_premium;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
