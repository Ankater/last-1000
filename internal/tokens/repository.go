package tokens

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
)

type Repository struct {
	DB *sql.DB
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (r *Repository) CreateToken(ctx context.Context, token string, name string) error {
	tokenHash := hashToken(token)

	_, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO tokens (token_hash, name) VALUES ($1, $2)`,
		tokenHash,
		name,
	)
	return err
}

func (r *Repository) TokenExists(ctx context.Context, token string) (bool, error) {
	tokenHash := hashToken(token)

	var exists bool

	err := r.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tokens WHERE token_hash = $1)", tokenHash).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
