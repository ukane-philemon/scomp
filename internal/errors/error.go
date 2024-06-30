package errors

// ErrorUnauthorized is the error for unauthorized requests.
type ErrorUnauthorized struct{}

func (eu *ErrorUnauthorized) Error() string {
	return "not authorized"
}
