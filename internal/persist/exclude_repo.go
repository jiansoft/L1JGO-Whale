package persist

import "context"

type ExcludeRepo struct {
	db *DB
}

func NewExcludeRepo(db *DB) *ExcludeRepo {
	return &ExcludeRepo{db: db}
}

// LoadByCharID returns all excluded names for a character.
func (r *ExcludeRepo) LoadByCharID(ctx context.Context, charID int32) ([]string, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT exclude_name FROM character_excludes WHERE char_id = $1`, charID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}
	return result, rows.Err()
}

// Add inserts an exclude entry.
func (r *ExcludeRepo) Add(ctx context.Context, charID int32, excludeName string) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO character_excludes (char_id, exclude_name) VALUES ($1, $2)
		 ON CONFLICT (char_id, exclude_name) DO NOTHING`,
		charID, excludeName,
	)
	return err
}

// Remove deletes an exclude entry (case-insensitive).
func (r *ExcludeRepo) Remove(ctx context.Context, charID int32, excludeName string) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM character_excludes WHERE char_id = $1 AND LOWER(exclude_name) = LOWER($2)`,
		charID, excludeName,
	)
	return err
}
