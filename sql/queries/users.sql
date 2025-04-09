-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByRefreshToken :one
WITH token_user AS (
    SELECT user_id FROM refresh_tokens WHERE token = $1
)
SELECT * FROM users WHERE id = (SELECT user_id FROM token_user);

-- name: UpdateUser :exec
UPDATE users SET email = $1, hashed_password = $2, updated_at = $3 WHERE id = $4;

-- name: UpgradeToPremium :exec
UPDATE users SET is_premium = TRUE WHERE id = $1;
