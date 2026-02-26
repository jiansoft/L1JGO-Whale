package persist

import "context"

// BuddyRow represents a persisted buddy list entry.
type BuddyRow struct {
	CharID    int32
	BuddyID   int32
	BuddyName string
}

type BuddyRepo struct {
	db *DB
}

func NewBuddyRepo(db *DB) *BuddyRepo {
	return &BuddyRepo{db: db}
}

// LoadByCharID returns all buddy entries for a character.
func (r *BuddyRepo) LoadByCharID(ctx context.Context, charID int32) ([]BuddyRow, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT char_id, buddy_id, buddy_name FROM character_buddys WHERE char_id = $1`, charID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []BuddyRow
	for rows.Next() {
		var b BuddyRow
		if err := rows.Scan(&b.CharID, &b.BuddyID, &b.BuddyName); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// Add inserts a new buddy entry.
func (r *BuddyRepo) Add(ctx context.Context, charID, buddyID int32, buddyName string) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO character_buddys (char_id, buddy_id, buddy_name) VALUES ($1, $2, $3)
		 ON CONFLICT (char_id, buddy_id) DO NOTHING`,
		charID, buddyID, buddyName,
	)
	return err
}

// Remove deletes a buddy entry by character ID and buddy name (case-insensitive).
func (r *BuddyRepo) Remove(ctx context.Context, charID int32, buddyName string) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM character_buddys WHERE char_id = $1 AND LOWER(buddy_name) = LOWER($2)`,
		charID, buddyName,
	)
	return err
}
