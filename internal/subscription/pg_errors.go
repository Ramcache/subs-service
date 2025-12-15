package subscription

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrConflict = errors.New("conflict")
)

func classifyPGError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrInvalidInput
		case "22P02":
			return ErrInvalidInput
		}
	}
	return err
}
