-- name: CreateRecipe :one
INSERT INTO recipes (
   id,
   user_id,
   created_at,
   updated_at,
   title,
   description,
   is_public,
   ingredients,
   instructions
) VALUES (
   $1, $2, NOW(), NOW(), $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetRecipeByID :one
SELECT * FROM recipes
WHERE id = $1
AND deleted_at IS NULL
AND (is_public = TRUE OR user_id = $2);

-- name: MarkRecipeSoftDeleteByID :exec
UPDATE recipes
SET deleted_at = NOW()
WHERE id = $1
AND user_id = $2
AND deleted_at IS NULL;

-- name: MarkAllUserRecipesSoftDelete :exec
UPDATE recipes
SET deleted_at = NOW()
WHERE user_id = $1
AND deleted_at IS NULL
AND is_public IS FALSE;

-- name: PurgeExpiredRecipes :exec
DELETE FROM recipes
WHERE deleted_at < NOW() - INTERVAL '30 days';

-- name: UpdateRecipeVisibility :one
UPDATE recipes
SET is_public = $1
WHERE user_id = $2
AND id = $3
AND deleted_at IS NULL
RETURNING *;

-- name: CompleteNutritionCalculation :one
UPDATE recipes
SET nutrition = $1,
   nutrition_status = 'completed',
   updated_at = NOW()
WHERE id = $2
   AND deleted_at IS NULL
RETURNING *;

-- name: UpdateRecipeFull :one
UPDATE recipes
SET updated_at = NOW(),
   title = $1,
   description = $2,
   is_public = $3,
   ingredients = $4,
   instructions = $5,
   nutrition = NULL,
   nutrition_status = 'unprocessed'
WHERE id = $6
AND user_id = $7
AND deleted_at IS NULL
RETURNING *;

-- name: GetAllPublicRecipesByOwnerID :many
SELECT 
   r.id,
   r.user_id,
   r.title,
   r.description,
   r.is_public,
   r.ingredients,
   r.instructions,
   r.created_at,
   r.updated_at,
   r.deleted_at,
   r.nutrition,
   r.nutrition_status,
   (
   CASE
      WHEN u.deleted_at IS NOT NULL or u.id IS NULL THEN 'Deleted User'
      ELSE u.display_name
   END
   )::TEXT AS author_name
FROM recipes r
JOIN users u ON r.user_id = u.id
WHERE user_id = $1
AND is_public IS TRUE
AND deleted_at IS NULL
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAllRecipesByOwnerID :many
SELECT 
   r.id,
   r.user_id,
   r.title,
   r.description,
   r.is_public,
   r.ingredients,
   r.instructions,
   r.created_at,
   r.updated_at,
   r.deleted_at,
   r.nutrition,
   r.nutrition_status,
   (
   CASE
      WHEN u.deleted_at IS NOT NULL or u.id IS NULL THEN 'Deleted User'
      ELSE u.display_name
   END
   )::TEXT AS author_name
FROM recipes r
JOIN users u ON r.user_id = u.id
WHERE user_id = $1
AND deleted_at IS NULL
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAllPublicRecipes :many
SELECT
   r.id,
   r.user_id,
   r.title,
   r.description,
   r.is_public,
   r.ingredients,
   r.instructions,
   r.created_at,
   r.updated_at,
   r.deleted_at,
   r.nutrition,
   r.nutrition_status,
   (
   CASE
      WHEN u.deleted_at IS NOT NULL or u.id IS NULL THEN 'Deleted User'
      ELSE u.display_name
   END
   )::TEXT AS author_name
FROM recipes r
JOIN users u ON r.user_id = u.id
WHERE r.is_public IS TRUE
AND r.deleted_at IS NULL
ORDER BY r.created_at DESC
LIMIT $1 OFFSET $2;
