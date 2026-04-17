package messages

import (
	"context"
	"database/sql"
	"time"
)

const pageSize = 100

type Message struct {
	ID        int64     `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) AddMessage(ctx context.Context, message string) error {
	_, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO messages (message) VALUES ($1)`,
		message,
	)
	return err
}

func (r *Repository) GetMessages(ctx context.Context, page int) ([]Message, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, message, created_at FROM messages ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`,
		pageSize, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Message, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
