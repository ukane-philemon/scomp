package db

import (
	"errors"
)

// RequiredClassSubjects is the number of required student subjects.
const RequiredClassSubjects = 10

var ErrorInvalidRequest = errors.New("invalid request")
