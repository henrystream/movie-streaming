-- name: RegisterUser :one
INSERT INTO users (email, password, membership_type)
VALUES ($1, $2, $3)
RETURNING id, email, membership_type;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;