-- name: GetLoadedEntryIds :many
SELECT id FROM entries ORDER BY created_at DESC;

-- name: LoadEntry :exec
INSERT INTO entries (id, text, creator_id, created_at, updated_at, embedding) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListEntriesByCosineSimilarity :many
SELECT id, text, CAST(1 - (embedding <=> $1) AS FLOAT(32)) AS cosine_similarity FROM entries ORDER BY cosine_similarity DESC limit $2;