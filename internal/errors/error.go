package errors

// ErrorUnauthorized is the error for unauthorized requests.
type ErrorUnauthorized struct{}

func (eu *ErrorUnauthorized) Error() string {
	return "not authorized"
}

// ErrorUnknown is a generic error sent for server related errors.
type ErrorUnknown struct{}

func (eu *ErrorUnknown) Error() string {
	return "Something unexpected happened, please try again later"
}
