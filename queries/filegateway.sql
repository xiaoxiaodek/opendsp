-- name: InsertFileRecord :exec
INSERT INTO file_record (id, namespace, storage_key, filename, size, content_type, status)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateFileRecordReady :exec
UPDATE file_record SET status = 2, size = $2, content_type = $3, md5 = $4, updated_at = NOW()
WHERE id = $1;

-- name: GetFileRecord :one
SELECT id, namespace, storage_key, filename, size, content_type, md5, status, created_at
FROM file_record WHERE id = $1;

-- name: DeleteFileRecord :exec
UPDATE file_record SET status = 3, updated_at = NOW() WHERE id = $1;
