package pgsql

import (
	"context"
)

// RetrieveSetDeclarations retrieves the list of declarations a set is associated with.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveSetDeclarations(ctx context.Context, setName string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT declaration_identifier FROM set_declarations WHERE set_name = $1;`,
		setName,
	)
}

// StoreSetDeclaration creates the association between a declaration and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) StoreSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
INSERT INTO set_declarations
    (declaration_identifier, set_name)
VALUES
    ($1, $2)
ON CONFLICT DO NOTHING;`,
		declarationID,
		setName,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveSetDeclaration removes the association between a declaration and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RemoveSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
DELETE FROM set_declarations
WHERE
    set_name = $1 AND
    declaration_identifier = $2;`,
		setName,
		declarationID,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveSets retrieves the list of sets.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveSets(ctx context.Context) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT DISTINCT set_name FROM set_declarations;`,
	)
}
