package apierr

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// IsNotFound returns true if the error is or wraps pgx.ErrNoRows.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
