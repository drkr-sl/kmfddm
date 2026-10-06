-- name: GetManifestItems :many
SELECT DISTINCT
    d.identifier,
    d.type,
    d.server_token
FROM
    declarations d
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name
WHERE
    es.enrollment_id = $1;

-- name: RemoveAllEnrollmentSets :execresult
DELETE FROM
    enrollment_sets
WHERE
    enrollment_id = $1;

-- name: GetDeclaration :one
SELECT
    d.identifier,
    d.type,
    d.payload,
    d.server_token,
    jsonb_build_object(
        'Identifier',  d.identifier,
        'Type',        d.type,
        'Payload',     d.payload,
        'ServerToken', d.server_token
    ) AS declaration
FROM
    declarations d
WHERE
    d.identifier = $1;

-- name: GetDDMDeclaration :one
SELECT
    jsonb_build_object(
        'Identifier',  d.identifier,
        'Type',        d.type,
        'Payload',     d.payload,
        'ServerToken', d.server_token
    ) AS declaration
FROM
    declarations d
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name
WHERE
    d.identifier = sqlc.arg(identifier) AND
    es.enrollment_id = sqlc.arg(enrollment_id) AND
    d.type LIKE sqlc.arg(type)
LIMIT 1;

-- name: RemoveDeclarationStatus :exec
DELETE FROM
    status_declarations
WHERE
    enrollment_id = $1;

-- name: PutDeclarationStatus :exec
INSERT INTO status_declarations (
    enrollment_id,
    item_type,
    declaration_identifier,
    active,
    valid,
    server_token,
    reasons,
    status_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetDeclarationStatus :many
SELECT
    sd.enrollment_id,
    sd.declaration_identifier,
    sd.active,
    sd.valid,
    sd.reasons,
    sd.server_token,
    sd.updated_at,
    sd.status_id,
    sd.server_token = COALESCE(d.server_token, '') AS current
FROM
    status_declarations sd
    LEFT JOIN declarations d
        ON sd.declaration_identifier = d.identifier
WHERE
    sd.enrollment_id = ANY(sqlc.arg(ids)::text[])
ORDER BY
    sd.enrollment_id;

-- name: InsertStatusError :exec
INSERT INTO status_errors (
    enrollment_id,
    path,
    error,
    status_id
) VALUES ($1, $2, $3, $4);

-- Keeps only the newest (offset) errors for the enrollment.
-- name: DeleteStatusErrors :exec
DELETE FROM
    status_errors
WHERE
    status_errors.enrollment_id = sqlc.arg(enrollment_id)
    AND status_errors.id <= (
        SELECT se.id FROM status_errors se
        WHERE se.enrollment_id = sqlc.arg(enrollment_id)
        ORDER BY se.id DESC
        LIMIT 1 OFFSET sqlc.arg(keep)
    );

-- name: SelectStatusErrors :many
SELECT
    enrollment_id,
    path,
    error,
    status_id,
    created_at
FROM
    status_errors
WHERE
    enrollment_id = ANY(sqlc.arg(ids)::text[])
ORDER BY
    enrollment_id, id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: InsertStatusReport :exec
INSERT INTO status_reports (
    enrollment_id,
    status_id,
    status_report
) VALUES ($1, $2, $3);

-- Keeps only the newest (offset) status reports for the enrollment.
-- name: DeleteStatusReports :exec
DELETE FROM
    status_reports
WHERE
    status_reports.enrollment_id = sqlc.arg(enrollment_id)
    AND status_reports.id <= (
        SELECT sr.id FROM status_reports sr
        WHERE sr.enrollment_id = sqlc.arg(enrollment_id)
        ORDER BY sr.id DESC
        LIMIT 1 OFFSET sqlc.arg(keep)
    );

-- Index 0 is the most recent status report for the enrollment.
-- name: SelectStatusReportByIndex :one
SELECT
    status_id,
    created_at,
    status_report
FROM
    status_reports
WHERE
    enrollment_id = sqlc.arg(enrollment_id)
ORDER BY
    id DESC
LIMIT 1 OFFSET sqlc.arg(row_offset);

-- name: SelectStatusReportByStatusID :one
SELECT
    sr.status_id,
    sr.created_at,
    sr.status_report,
    (
        SELECT COUNT(*) FROM status_reports
        WHERE status_reports.enrollment_id = sr.enrollment_id
            AND status_reports.id > sr.id
    ) AS idx
FROM
    status_reports sr
WHERE
    sr.enrollment_id = $1
    AND sr.status_id = $2
ORDER BY
    sr.id DESC
LIMIT 1;
