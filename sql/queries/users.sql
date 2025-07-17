-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3
)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserById :one
SELECT * FROM users 
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users 
WHERE email = $1;
