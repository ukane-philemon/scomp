package db

import (
	"errors"
)

// RequiredClassSubjects is the number of required student subjects.
const RequiredClassSubjects = 10

// ErrorInvalidRequest is a user facing error returned by repositories.
var ErrorInvalidRequest = errors.New("invalid request")
