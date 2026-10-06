package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

// StoreDeclaration stores a declaration and returns whether it changed or not.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error) {
	// the WHERE on the conflict update leaves an unchanged declaration
	// untouched, so it reports no affected rows and keeps its server token.
	result, err := s.db.ExecContext(
		ctx,
		`
INSERT INTO declarations
    (identifier, type, payload, server_token)
VALUES
    ($1::text, $2::text, $3::jsonb, encode(sha256(convert_to(concat($1::text, $2::text, $3::jsonb::text, CURRENT_TIMESTAMP::text, '0'), 'UTF8')), 'hex'))
ON CONFLICT (identifier) DO UPDATE
SET
    type         = excluded.type,
    payload      = excluded.payload,
    server_token = encode(sha256(convert_to(concat(excluded.identifier, excluded.type, excluded.payload::text, declarations.created_at::text, declarations.touched_ct::text), 'UTF8')), 'hex'),
    updated_at   = CURRENT_TIMESTAMP
WHERE
    declarations.type IS DISTINCT FROM excluded.type OR
    declarations.payload IS DISTINCT FROM excluded.payload;`,
		d.Identifier,
		d.Type,
		string(d.Payload),
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveDeclaration retrieves a declaration.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error) {
	qd, err := s.q.GetDeclaration(ctx, declarationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	} else if err != nil {
		return nil, err
	}
	return &ddm.Declaration{
		Identifier:  qd.Identifier,
		Type:        qd.Type,
		ServerToken: qd.ServerToken,
		Payload:     []byte(qd.Payload),
		Raw:         []byte(qd.Declaration),
	}, nil
}

// RetrieveDeclarationModTime retrieves the last modification time of the declaration.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveDeclarationModTime(ctx context.Context, declarationID string) (time.Time, error) {
	var updatedAt time.Time
	if err := s.db.QueryRowContext(ctx, "SELECT updated_at FROM declarations WHERE identifier = $1 LIMIT 1;", declarationID).Scan(&updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
		}
		return time.Time{}, err
	}
	return updatedAt, nil
}

// DeleteDeclaration deletes a declaration and returns whether it was deleted or already existed.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) DeleteDeclaration(ctx context.Context, declarationID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM declarations WHERE identifier = $1;`,
		declarationID,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveDeclarationSets returns the list of sets a declaration is a part of.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveDeclarationSets(ctx context.Context, declarationID string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT set_name FROM set_declarations WHERE declaration_identifier = $1;`,
		declarationID,
	)
}

// RetrieveDeclarations returns the list of declaration IDs.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveDeclarations(ctx context.Context) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT identifier FROM declarations;`,
	)
}

// TouchDeclaration updates a declaration's "touch count" which makes a new server token.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) TouchDeclaration(ctx context.Context, declarationID string) error {
	result, err := s.db.ExecContext(
		ctx,
		`
UPDATE
    declarations
SET
    touched_ct   = touched_ct + 1,
    server_token = encode(sha256(convert_to(concat(identifier, type, payload::text, created_at::text, (touched_ct + 1)::text), 'UTF8')), 'hex'),
    updated_at   = CURRENT_TIMESTAMP
WHERE
    identifier = $1;`,
		declarationID,
	)
	if err != nil {
		return err
	}
	changed, err := resultChangedRows(result)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("%w: declaration not touched (may not exist)", storage.ErrDeclarationNotFound)
	}
	return nil
}
