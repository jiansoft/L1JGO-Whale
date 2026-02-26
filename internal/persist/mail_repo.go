package persist

import (
	"context"
	"time"
)

// MailRow represents a single mail record.
type MailRow struct {
	ID         int32
	Type       int16
	Sender     string
	Receiver   string
	Date       time.Time
	ReadStatus int16
	InboxID    int32
	Subject    []byte
	Content    []byte
}

type MailRepo struct {
	db *DB
}

func NewMailRepo(db *DB) *MailRepo {
	return &MailRepo{db: db}
}

// LoadByInbox returns all mails for a given inbox owner and mail type.
func (r *MailRepo) LoadByInbox(ctx context.Context, inboxID int32, mailType int16) ([]MailRow, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, type, sender, receiver, date, read_status, inbox_id, subject, content
		 FROM mail WHERE inbox_id = $1 AND type = $2 ORDER BY id DESC`,
		inboxID, mailType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MailRow
	for rows.Next() {
		var m MailRow
		if err := rows.Scan(&m.ID, &m.Type, &m.Sender, &m.Receiver, &m.Date,
			&m.ReadStatus, &m.InboxID, &m.Subject, &m.Content); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetByID returns a single mail by ID, or nil if not found.
func (r *MailRepo) GetByID(ctx context.Context, id int32) (*MailRow, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT id, type, sender, receiver, date, read_status, inbox_id, subject, content
		 FROM mail WHERE id = $1`, id)

	var m MailRow
	err := row.Scan(&m.ID, &m.Type, &m.Sender, &m.Receiver, &m.Date,
		&m.ReadStatus, &m.InboxID, &m.Subject, &m.Content)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// Write inserts a new mail record and returns the generated ID.
func (r *MailRepo) Write(ctx context.Context, m *MailRow) (int32, error) {
	var id int32
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO mail (type, sender, receiver, date, read_status, inbox_id, subject, content)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		m.Type, m.Sender, m.Receiver, m.Date, m.ReadStatus, m.InboxID, m.Subject, m.Content,
	).Scan(&id)
	return id, err
}

// SetReadStatus marks a mail as read (read_status = 1).
func (r *MailRepo) SetReadStatus(ctx context.Context, id int32) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE mail SET read_status = 1 WHERE id = $1`, id)
	return err
}

// SetType changes the type of a mail (e.g., move to storage box).
func (r *MailRepo) SetType(ctx context.Context, id int32, newType int16) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE mail SET type = $1 WHERE id = $2`, newType, id)
	return err
}

// Delete removes a mail by ID.
func (r *MailRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM mail WHERE id = $1`, id)
	return err
}

// CountByInbox returns the number of mails for a given inbox owner and type.
func (r *MailRepo) CountByInbox(ctx context.Context, inboxID int32, mailType int16) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM mail WHERE inbox_id = $1 AND type = $2`,
		inboxID, mailType).Scan(&count)
	return count, err
}
