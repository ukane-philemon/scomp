package auth

type Repository interface {
	// GenerateToken generates a new auth token for uniqueID.
	GenerateToken(uniqueID string) (string, error)
	// IsValid checks the token is valid and return it's uniqueID.
	IsValid(token string) (string, bool)
}
