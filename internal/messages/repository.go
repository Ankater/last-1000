package messages

import (
	"context"
	"database/sql"
)

type Repository struct {
	DB *sql.DB
}

func (r Repository) AddMessage(ctx context.Context, message string) error {
	_, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO messages (message) VALUES ($1)`,
		message,
	)

	return err
}
