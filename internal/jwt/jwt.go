package jwt

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/cristalhq/jwt/v4"
)

const (
	jwtIssuer = "SCOMP"

	JWTExpiry        = 24 * time.Hour
	jwtAudienceAdmin = "admin"
	jwtAlg           = jwt.HS256
)

type Manager struct {
	aud      string
	builder  *jwt.Builder
	verifier jwt.Verifier
}

// NewJWTManager returns a new manager for jwt tokens.
func NewJWTManager() (*Manager, error) {
	jwtSecret := make([]byte, 32)
	_, err := rand.Read(jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("rand.Read error: %w", err)
	}

	signer, err := jwt.NewSignerHS(jwtAlg, jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("jwt.NewSignerHS error: %w", err)
	}

	verifier, err := jwt.NewVerifierHS(jwtAlg, jwtSecret[:])
	if err != nil {
		return nil, fmt.Errorf("jwt.NewVerifierHS error: %w", err)
	}

	m := &Manager{
		aud:      jwtAudienceAdmin,
		builder:  jwt.NewBuilder(signer),
		verifier: verifier,
	}

	return m, nil
}

// GenerateJWtToken generates a new jwt token for the specified id.
func (m *Manager) GenerateJWtToken(id string) (string, error) {
	claims := &jwt.RegisteredClaims{
		ID:        id,
		Audience:  jwt.Audience{jwtAudienceAdmin},
		Issuer:    jwtIssuer,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTExpiry)),
	}

	token, err := m.builder.Build(claims)
	if err != nil {
		return "", fmt.Errorf("m.builder.Build error: %w", err)
	}

	return token.String(), nil
}

// IsValidToken checks that the provided token is valid and returns the unique
// id added to the auth token.
func (m *Manager) IsValidToken(jwtToken string) (string, bool) {
	jwtClaims := new(jwt.RegisteredClaims)
	err := jwt.ParseClaims([]byte(jwtToken), m.verifier, jwtClaims)
	if err != nil || !(jwtClaims.IsIssuer(jwtIssuer) && jwtClaims.IsValidAt(time.Now())) || !jwtClaims.IsForAudience(m.aud) {
		return "", false
	}

	return jwtClaims.ID, true
}
