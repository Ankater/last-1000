package tokens

import (
	"context"
	"fmt"
)

type tokenCreator interface {
	CreateToken(ctx context.Context, token string, name string) error
}

type Service struct {
	Repo tokenCreator
}

func (s *Service) CreateToken(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	token := GenerateToken(32)

	if err := s.Repo.CreateToken(ctx, token, name); err != nil {
		return "", fmt.Errorf("create token %q: %w", name, err)
	}

	return token, nil
}
