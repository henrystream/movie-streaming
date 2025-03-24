-- name: CreateMovie :one
INSERT INTO movies (title, genre, year)
VALUES ($1, $2, $3)
RETURNING id, title, genre, year;

-- name: GetMovie :one
SELECT id, title, genre, year FROM movies WHERE id = $1;

-- name: ListMovies :many
SELECT id, title, genre, year FROM movies;

-- name: UpdateMovie :one
UPDATE movies SET title = $2, genre = $3, year = $4
WHERE id = $1
RETURNING id, title, genre, year;

-- name: DeleteMovie :execrows
DELETE FROM movies WHERE id = $1;