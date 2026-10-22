package pgw

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The wrappers embed the pgx interface, so anything not overridden passes through untouched.

type row struct {
	pgx.Row
	p pgErrorParser
}

func (r row) Scan(dest ...any) error { return r.p.ParsePgError(r.Row.Scan(dest...)) }

type rows struct {
	pgx.Rows
	p pgErrorParser
}

func (r rows) Err() error { return r.p.ParsePgError(r.Rows.Err()) }

type batchResults struct {
	pgx.BatchResults
	p pgErrorParser
}

func (b batchResults) Exec() (pgconn.CommandTag, error) {
	tag, err := b.BatchResults.Exec()
	return tag, b.p.ParsePgError(err)
}

func (b batchResults) Query() (pgx.Rows, error) {
	r, err := b.BatchResults.Query()
	return wrapRows(r, b.p), b.p.ParsePgError(err)
}

func (b batchResults) QueryRow() pgx.Row { return row{b.BatchResults.QueryRow(), b.p} }

func (b batchResults) Close() error { return b.p.ParsePgError(b.BatchResults.Close()) }

type tx struct {
	pgx.Tx
	p pgErrorParser
}

// wrapTx returns a nil interface (not a typed nil) when the underlying Begin failed.
func wrapTx(t pgx.Tx, p pgErrorParser, err error) (pgx.Tx, error) {
	if err != nil {
		return nil, p.ParsePgError(err)
	}
	return tx{t, p}, nil
}

func wrapRows(r pgx.Rows, p pgErrorParser) pgx.Rows {
	if r == nil {
		return nil
	}
	return rows{r, p}
}

func (t tx) Begin(ctx context.Context) (pgx.Tx, error) {
	nested, err := t.Tx.Begin(ctx)
	return wrapTx(nested, t.p, err)
}

func (t tx) Commit(ctx context.Context) error { return t.p.ParsePgError(t.Tx.Commit(ctx)) }

func (t tx) Rollback(ctx context.Context) error { return t.p.ParsePgError(t.Tx.Rollback(ctx)) }

func (t tx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	n, err := t.Tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	return n, t.p.ParsePgError(err)
}

func (t tx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return batchResults{t.Tx.SendBatch(ctx, b), t.p}
}

func (t tx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	sd, err := t.Tx.Prepare(ctx, name, sql)
	return sd, t.p.ParsePgError(err)
}

func (t tx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tag, err := t.Tx.Exec(ctx, sql, args...)
	return tag, t.p.ParsePgError(err)
}

func (t tx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	r, err := t.Tx.Query(ctx, sql, args...)
	return wrapRows(r, t.p), t.p.ParsePgError(err)
}

func (t tx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return row{t.Tx.QueryRow(ctx, sql, args...), t.p}
}
