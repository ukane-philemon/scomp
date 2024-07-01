package auth

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

// AuthRepository implements Repository.
type AuthRepository struct {
	aud      string
	builder  *jwt.Builder
	verifier jwt.Verifier
}

// NewRepository returns a new  instance of *AuthRepository.
func NewRepository() (Repository, error) {
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

	return &AuthRepository{
		aud:      jwtAudienceAdmin,
		builder:  jwt.NewBuilder(signer),
		verifier: verifier,
	}, nil
}

// GenerateToken generates a new auth token for uniqueID.
// Implements Repository.
func (ar *AuthRepository) GenerateToken(uniqueID string) (string, error) {
	claims := &jwt.RegisteredClaims{
		ID:        uniqueID,
		Audience:  jwt.Audience{jwtAudienceAdmin},
		Issuer:    jwtIssuer,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTExpiry)),
	}

	token, err := ar.builder.Build(claims)
	if err != nil {
		return "", fmt.Errorf("m.builder.Build error: %w", err)
	}

	return token.String(), nil
}

// IsValid checks the token is valid and return it's uniqueID.
// Implements Repository.
func (ar *AuthRepository) IsValid(jwtToken string) (string, bool) {
	jwtClaims := new(jwt.RegisteredClaims)
	err := jwt.ParseClaims([]byte(jwtToken), ar.verifier, jwtClaims)
	if err != nil || !(jwtClaims.IsIssuer(jwtIssuer) && jwtClaims.IsValidAt(time.Now())) || !jwtClaims.IsForAudience(ar.aud) {
		return "", false
	}

	return jwtClaims.ID, true
}
