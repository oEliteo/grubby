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

-- name: UpdateUserPartial :one
UPDATE users
SET display_name = COALESCE(sqlc.narg('display_name'), display_name),
   email = COALESCE(sqlc.narg('email'), email),
   hashed_password = COALESCE(sqlc.narg('hashed_password'), hashed_password),
   updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, created_at, updated_at, display_name, email, is_premium;

-- name: UpdateUserFull :one
UPDATE users
SET display_name = $1,
   email = $2,
   hashed_password = $3,
   updated_at = NOW()
WHERE id = $4
RETURNING id, created_at, updated_at, display_name, email, is_premium;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
