package admin

type Repository interface {
	// CreateAccount creates a new admin and returns their id.
	CreateAccount(username, password string) (string, error)
	// LoginAccount authenticate and admin and returns their id.
	LoginAccount(username, password string) (string, error)
}
