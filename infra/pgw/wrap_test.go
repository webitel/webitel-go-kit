package pgw

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errDomain = errors.New("domain")

func uniqueErr() error {
	return &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key", TableName: "users"}
}

func parser() *errorsManager {
	m := newErrorsManager()
	_ = m.RegisterUniqueViolation("users_email_key", func(*pgconn.PgError) error { return errDomain })
	return m
}

func TestParsePgError(t *testing.T) {
	m := newErrorsManager()

	if m.ParsePgError(nil) != nil {
		t.Fatal("nil must stay nil")
	}
	for _, err := range []error{pgx.ErrNoRows, context.Canceled, pgx.ErrTxClosed, fmt.Errorf("w: %w", pgx.ErrNoRows)} {
		if got := m.ParsePgError(err); got != err {
			t.Fatalf("%v must pass through, got %v", err, got)
		}
	}

	for _, code := range []string{"23502", "23503", "23505", "23514"} {
		src := &pgconn.PgError{Code: code, ConstraintName: "c", TableName: "t", ColumnName: "col"}
		var de *DataExceptionError
		if !errors.As(m.ParsePgError(fmt.Errorf("wrapped: %w", src)), &de) {
			t.Fatalf("%s: want DataExceptionError", code)
		}
		if de.Code != code || de.ConstraintName != "c" {
			t.Fatalf("%s: Code/ConstraintName lost: %+v", code, de)
		}
		var pg *pgconn.PgError
		if !errors.As(de, &pg) || pg != src {
			t.Fatalf("%s: Unwrap must return the original PgError", code)
		}
	}

	for _, code := range []string{"08000", "08001", "08006", "08P01"} {
		var ce *ConnectionExceptionError
		src := &pgconn.PgError{Code: code}
		if !errors.As(m.ParsePgError(src), &ce) || ce.Code != code {
			t.Fatalf("%s: want ConnectionExceptionError", code)
		}
		var pg *pgconn.PgError
		if !errors.As(ce, &pg) || pg != src {
			t.Fatalf("%s: Unwrap must return the original PgError", code)
		}
	}

	for _, code := range []string{"23P01", "40001", "22001"} {
		src := &pgconn.PgError{Code: code}
		if m.ParsePgError(src) != error(src) {
			t.Fatalf("%s must pass through", code)
		}
	}

	// a second pass must not wrap the already mapped error again
	once := m.ParsePgError(uniqueErr())
	if m.ParsePgError(once) != once {
		t.Fatal("double mapping")
	}
}

func TestUnwrapOfHandmadeError(t *testing.T) {
	var pg *pgconn.PgError
	if errors.As(&DataExceptionError{}, &pg) || errors.As(&ConnectionExceptionError{}, &pg) {
		t.Fatal("an error created by hand has no PgError to unwrap")
	}
}

func TestProcessorWins(t *testing.T) {
	if got := parser().ParsePgError(uniqueErr()); got != errDomain {
		t.Fatalf("want processor result, got %v", got)
	}
}

type fakeRow struct{ err error }

func (r fakeRow) Scan(...any) error { return r.err }

type fakeRows struct {
	pgx.Rows
	err error
}

func (r fakeRows) Err() error { return r.err }

type fakeBatch struct {
	pgx.BatchResults
	err error
}

func (b fakeBatch) Exec() (pgconn.CommandTag, error) { return pgconn.CommandTag{}, b.err }
func (b fakeBatch) Close() error                     { return b.err }
func (b fakeBatch) QueryRow() pgx.Row                { return fakeRow{b.err} }

type fakeTx struct {
	pgx.Tx
	err error
}

func (f fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, f.err
}
func (f fakeTx) QueryRow(context.Context, string, ...any) pgx.Row { return fakeRow{f.err} }
func (f fakeTx) Commit(context.Context) error                     { return f.err }
func (f fakeTx) Begin(context.Context) (pgx.Tx, error)            { return fakeTx{err: f.err}, nil }

func TestWrappers(t *testing.T) {
	p := parser()

	if err := (row{fakeRow{uniqueErr()}, p}).Scan(); err != errDomain {
		t.Fatalf("row.Scan: %v", err)
	}
	if err := (row{fakeRow{pgx.ErrNoRows}, p}).Scan(); err != pgx.ErrNoRows {
		t.Fatalf("row.Scan must keep ErrNoRows: %v", err)
	}

	// pgx.CollectRows reports the error from rows.Err()
	if err := (rows{fakeRows{err: uniqueErr()}, p}).Err(); err != errDomain {
		t.Fatalf("rows.Err: %v", err)
	}

	b := batchResults{fakeBatch{err: uniqueErr()}, p}
	if _, err := b.Exec(); err != errDomain {
		t.Fatalf("batch.Exec: %v", err)
	}
	if err := b.QueryRow().Scan(); err != errDomain {
		t.Fatalf("batch.QueryRow: %v", err)
	}
	if err := b.Close(); err != errDomain {
		t.Fatalf("batch.Close: %v", err)
	}

	tr, err := wrapTx(fakeTx{err: uniqueErr()}, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := tr.Exec(ctx, ""); err != errDomain {
		t.Fatalf("tx.Exec: %v", err)
	}
	if err := tr.QueryRow(ctx, "").Scan(); err != errDomain {
		t.Fatalf("tx.QueryRow: %v", err)
	}
	if err := tr.Commit(ctx); err != errDomain {
		t.Fatalf("tx.Commit: %v", err)
	}
	nested, _ := tr.Begin(ctx) // savepoint keeps the mapping
	if err := nested.Commit(ctx); err != errDomain {
		t.Fatalf("savepoint commit: %v", err)
	}

	// a failed Begin must give a nil interface, not a typed nil
	if got, err := wrapTx(nil, p, uniqueErr()); got != nil || err != errDomain {
		t.Fatalf("wrapTx on error: %v, %v", got, err)
	}
}

// Pool must hand out the wrappers; the pgxpool never connects here (nothing listens on port 1).
func TestPoolReturnsWrappers(t *testing.T) {
	ctx := context.Background()
	pp, err := pgxpool.New(ctx, "postgres://u:p@127.0.0.1:1/db?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pp.Close()
	p, err := newPool(pp, PoolConfig{ErrorParser: newErrorsManager()})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := p.QueryRow(ctx, "select 1").(row); !ok {
		t.Fatal("QueryRow is not wrapped")
	}
	if _, ok := p.SendBatch(ctx, &pgx.Batch{}).(batchResults); !ok {
		t.Fatal("SendBatch is not wrapped")
	}
	if r, _ := p.Query(ctx, "select 1"); r != nil {
		if _, ok := r.(rows); !ok {
			t.Fatal("Query is not wrapped")
		}
	}
	if got, err := p.Begin(ctx); got != nil || err == nil {
		t.Fatalf("Begin must return a nil Tx and an error, got %v, %v", got, err)
	}
}
