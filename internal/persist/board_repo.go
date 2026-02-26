package persist

import "context"

// BoardPost represents a single bulletin board post.
type BoardPost struct {
	ID      int32
	Name    string // author character name
	Date    string // "yyyy/MM/dd"
	Title   string
	Content string
}

type BoardRepo struct {
	db *DB
}

func NewBoardRepo(db *DB) *BoardRepo {
	return &BoardRepo{db: db}
}

// ListPage returns up to `limit` posts with ID <= beforeID, ordered newest first.
// If beforeID <= 0, returns the newest posts (no upper bound).
// Note: Java uses inclusive (id <= number) — the boundary post appears on both pages.
// The 3.80C client expects this overlap behavior.
func (r *BoardRepo) ListPage(ctx context.Context, beforeID int32, limit int) ([]BoardPost, error) {
	var query string
	var args []interface{}

	if beforeID <= 0 {
		query = `SELECT id, name, date, title FROM server_board ORDER BY id DESC LIMIT $1`
		args = []interface{}{limit}
	} else {
		query = `SELECT id, name, date, title FROM server_board WHERE id <= $1 ORDER BY id DESC LIMIT $2`
		args = []interface{}{beforeID, limit}
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []BoardPost
	for rows.Next() {
		var p BoardPost
		if err := rows.Scan(&p.ID, &p.Name, &p.Date, &p.Title); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// GetByID returns a single post with full content, or nil if not found.
func (r *BoardRepo) GetByID(ctx context.Context, id int32) (*BoardPost, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT id, name, date, title, content FROM server_board WHERE id = $1`, id)

	var p BoardPost
	err := row.Scan(&p.ID, &p.Name, &p.Date, &p.Title, &p.Content)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// Write inserts a new board post and returns the generated ID.
func (r *BoardRepo) Write(ctx context.Context, name, date, title, content string) (int32, error) {
	var id int32
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO server_board (name, date, title, content) VALUES ($1, $2, $3, $4) RETURNING id`,
		name, date, title, content,
	).Scan(&id)
	return id, err
}

// Delete removes a board post by ID.
func (r *BoardRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM server_board WHERE id = $1`, id)
	return err
}
