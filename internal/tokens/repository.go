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
