-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3
)
RETURNING *;

-- name: DeleteAllChirps :exec
DELETE FROM chirps;

-- name: GetAllChirps :many
SELECT * FROM chirps
ORDER BY created_at ASC;

-- name: GetChirpById :one
SELECT * FROM chirps
WHERE id = $1;
