package graph

import (
	"errors"
	"log"

	"github.com/ukane-philemon/scomp/internal/db"
	customerror "github.com/ukane-philemon/scomp/internal/errors"
)

// handleError checks if err is a server error and logs if before returning a
// special error.
func handleError(err error) error {
	if errors.Is(err, db.ErrorInvalidRequest) {
		return err
	}

	log.Printf("SERVER ERROR: %v", err.Error())
	return &customerror.ErrorUnknown{}
}
