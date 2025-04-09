-- name: CreateZinger :one
INSERT INTO zingers (id, created_at, updated_at, body, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetAllZingers :many
SELECT * FROM zingers ORDER BY created_at ASC;

-- name: GetZingersByUser :many
SELECT * FROM zingers WHERE user_id = $1 ORDER BY created_at ASC;

-- name: GetZingerById :one
SELECT * FROM zingers WHERE id = $1;

-- name: DeleteZingerById :exec
DELETE FROM zingers WHERE id = $1;