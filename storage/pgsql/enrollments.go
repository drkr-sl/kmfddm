package pgsql

import (
	"context"
	"errors"

	"github.com/lib/pq"
)

// RetrieveEnrollmentSets retrieves the list of sets an enrollment is assigned to.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveEnrollmentSets(ctx context.Context, enrollmentID string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT set_name FROM enrollment_sets WHERE enrollment_id = $1;`,
		enrollmentID,
	)
}

// StoreEnrollmentSet creates the association between an enrollment and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) StoreEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
INSERT INTO enrollment_sets
    (enrollment_id, set_name)
VALUES
    ($1, $2)
ON CONFLICT DO NOTHING;`,
		enrollmentID,
		setName,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveEnrollmentSet removes the association between an enrollment and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RemoveEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
DELETE FROM enrollment_sets
WHERE
    enrollment_id = $1 AND
    set_name = $2;`,
		enrollmentID,
		setName,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveAllEnrollmentSets dissociates enrollment ID from any sets.
// If any associations are removed true is returned.
// It should not be an error if no associations exist.
func (s *PSQLStorage) RemoveAllEnrollmentSets(ctx context.Context, enrollmentID string) (bool, error) {
	r, err := s.q.RemoveAllEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		return false, err
	}
	return resultChangedRows(r)
}

// RetrieveEnrollmentIDs retrieves enrollment IDs.
// See also the storage package for documentation on the storage interfaces.
func (s *PSQLStorage) RetrieveEnrollmentIDs(ctx context.Context, declarations []string, sets []string, ids []string) ([]string, error) {
	if len(declarations) < 1 && len(sets) < 1 && len(ids) < 1 {
		return nil, errors.New("no parameters provided")
	}
	// an empty array matches nothing, so unused filters drop out of the OR.
	rows, err := s.db.QueryContext(
		ctx, `
SELECT DISTINCT
    es.enrollment_id
FROM
    enrollment_sets es
    LEFT JOIN set_declarations sd
        ON sd.set_name = es.set_name
    LEFT JOIN declarations d
        ON d.identifier = sd.declaration_identifier
WHERE
    d.identifier = ANY($1) OR
    es.set_name = ANY($2) OR
    es.enrollment_id = ANY($3);`,
		pq.Array(declarations),
		pq.Array(sets),
		pq.Array(ids),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	retIDMap := make(map[string]struct{})
	var retID string
	for rows.Next() {
		err = rows.Scan(&retID)
		if err != nil {
			break
		}
		retIDMap[retID] = struct{}{}
	}
	if err == nil {
		err = rows.Err()
	}
	// merge in the enrollment IDs directly supplied in params
	for _, id := range ids {
		retIDMap[id] = struct{}{}
	}
	// convert back to slice for return value
	retIDs := make([]string, 0, len(retIDMap))
	for k := range retIDMap {
		retIDs = append(retIDs, k)
	}
	return retIDs, err
}
