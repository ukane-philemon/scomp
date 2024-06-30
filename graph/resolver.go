package graph

import (
	"fmt"

	"github.com/ukane-philemon/scomp/internal/jwt"
)

type Resolver struct {
	db         ClassDatabase
	JWTManager *jwt.Manager
}

// NewResolver creates and returns a new instance of *Resolver.
func NewResolver(db ClassDatabase) (*Resolver, error) {
	jwtManager, err := jwt.NewJWTManager()
	if err != nil {
		return nil, fmt.Errorf("jwt.NewJWTManager error: %w", err)
	}

	return &Resolver{
		db:         db,
		JWTManager: jwtManager,
	}, nil
}
