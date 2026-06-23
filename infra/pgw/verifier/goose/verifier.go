package goose

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/webitel-go-kit/infra/pgw"
)

func NewGooseMigrationVerifier(fromTable string, minVersionRequired string) (pgw.MigrationVerifier, error) {
	if fromTable == "" {
		return nil, errors.New("versioning table required during verification of goose migration")
	}
	if minVersionRequired == "" {
		return nil, errors.New("min version parameter required during verification of goose migration")
	}
	return func(ctx context.Context, conn *pgxpool.Conn) error {
		query := fmt.Sprintf("SELECT true FROM %s WHERE version_id = $1 AND is_applied = true", fromTable)

		res, err := conn.Exec(ctx, query, minVersionRequired)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return errors.New("goose migration verification failed: no matching version found")
		}
		return nil

	}, nil

}
