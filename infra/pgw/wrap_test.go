package pgw

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"

	werror "github.com/webitel/webitel-go-kit/pkg/errors"
)

var errDomain = errors.New("domain")

func uniqueErr() error {
	return &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "users_email_key", TableName: "users"}
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

	tests := []struct {
		name       string
		in         error
		wantCode   codes.Code
		wantID     string
		wantIs     error
		wantValues map[string]string
	}{
		{
			name:     "no rows is not found",
			in:       pgx.ErrNoRows,
			wantCode: codes.NotFound,
			wantID:   "pgw.not_found",
			wantIs:   ErrNotFound,
		},
		{
			name:     "wrapped no rows is not found",
			in:       fmt.Errorf("scan: %w", pgx.ErrNoRows),
			wantCode: codes.NotFound,
			wantID:   "pgw.not_found",
			wantIs:   ErrNotFound,
		},
		{
			name:     "context canceled",
			in:       context.Canceled,
			wantCode: codes.Canceled,
			wantID:   "pgw.canceled",
			wantIs:   ErrCanceled,
		},
		{
			name:     "context deadline exceeded",
			in:       context.DeadlineExceeded,
			wantCode: codes.DeadlineExceeded,
			wantID:   "pgw.deadline_exceeded",
			wantIs:   ErrDeadlineExceeded,
		},
		{
			name:     "unique violation",
			in:       &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "space_name_uq"},
			wantCode: codes.AlreadyExists,
			wantID:   "pgw.unique",
			wantIs:   ErrUnique,
			wantValues: map[string]string{
				"constraint": "space_name_uq",
			},
		},
		{
			name: "foreign key violation",
			in: &pgconn.PgError{
				Code:           pgerrcode.ForeignKeyViolation,
				ConstraintName: "space_model_fk",
				TableName:      "space",
			},
			wantCode: codes.FailedPrecondition,
			wantID:   "pgw.foreign_key",
			wantIs:   ErrForeignKey,
			wantValues: map[string]string{
				"constraint": "space_model_fk",
				"table":      "space",
			},
		},
		{
			name:     "check violation",
			in:       &pgconn.PgError{Code: pgerrcode.CheckViolation, ConstraintName: "dim_check"},
			wantCode: codes.InvalidArgument,
			wantID:   "pgw.check",
			wantIs:   ErrCheck,
			wantValues: map[string]string{
				"constraint": "dim_check",
			},
		},
		{
			name: "not null violation",
			in: &pgconn.PgError{
				Code:       pgerrcode.NotNullViolation,
				TableName:  "space",
				ColumnName: "name",
			},
			wantCode: codes.InvalidArgument,
			wantID:   "pgw.not_null",
			wantIs:   ErrNotNull,
			wantValues: map[string]string{
				"table":  "space",
				"column": "name",
			},
		},
		{
			name: "class 22 data exception is invalid argument",
			in: &pgconn.PgError{
				Code:       pgerrcode.InvalidTextRepresentation,
				SchemaName: "ai_hub",
				TableName:  "space",
				ColumnName: "id",
			},
			wantCode: codes.InvalidArgument,
			wantID:   "pgw.data",
			wantIs:   ErrData,
			wantValues: map[string]string{
				"schema": "ai_hub",
				"table":  "space",
				"column": "id",
			},
		},
		{
			name:     "connection exception is unavailable",
			in:       &pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			wantCode: codes.Unavailable,
			wantID:   "pgw.connection",
			wantIs:   ErrConnection,
			wantValues: map[string]string{
				"sqlstate": pgerrcode.ConnectionFailure,
			},
		},
		{
			name:     "admin shutdown is unavailable",
			in:       &pgconn.PgError{Code: pgerrcode.AdminShutdown},
			wantCode: codes.Unavailable,
			wantID:   "pgw.connection",
			wantIs:   ErrConnection,
			wantValues: map[string]string{
				"sqlstate": pgerrcode.AdminShutdown,
			},
		},
		{
			name:     "crash shutdown is unavailable",
			in:       &pgconn.PgError{Code: pgerrcode.CrashShutdown},
			wantCode: codes.Unavailable,
			wantID:   "pgw.connection",
			wantIs:   ErrConnection,
			wantValues: map[string]string{
				"sqlstate": pgerrcode.CrashShutdown,
			},
		},
		{
			name:     "cannot connect now is unavailable",
			in:       &pgconn.PgError{Code: pgerrcode.CannotConnectNow},
			wantCode: codes.Unavailable,
			wantID:   "pgw.connection",
			wantIs:   ErrConnection,
			wantValues: map[string]string{
				"sqlstate": pgerrcode.CannotConnectNow,
			},
		},
		{
			name:     "read only sql transaction is unavailable",
			in:       &pgconn.PgError{Code: pgerrcode.ReadOnlySQLTransaction},
			wantCode: codes.Unavailable,
			wantID:   "pgw.connection",
			wantIs:   ErrConnection,
			wantValues: map[string]string{
				"sqlstate": pgerrcode.ReadOnlySQLTransaction,
			},
		},
		{
			name:     "serialization failure is aborted",
			in:       &pgconn.PgError{Code: pgerrcode.SerializationFailure},
			wantCode: codes.Aborted,
			wantID:   "pgw.aborted",
			wantIs:   ErrAborted,
		},
		{
			name:     "deadlock is aborted",
			in:       &pgconn.PgError{Code: pgerrcode.DeadlockDetected},
			wantCode: codes.Aborted,
			wantID:   "pgw.aborted",
			wantIs:   ErrAborted,
		},
		{
			name:     "program limit exceeded is invalid argument",
			in:       &pgconn.PgError{Code: pgerrcode.ProgramLimitExceeded},
			wantCode: codes.InvalidArgument,
			wantID:   "pgw.program_limit",
			wantIs:   ErrProgramLimit,
		},
		{
			name:     "unknown pg error is internal",
			in:       &pgconn.PgError{Code: pgerrcode.InternalError},
			wantCode: codes.Internal,
			wantID:   "pgw.internal",
			wantIs:   ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.ParsePgError(tt.in)
			if got == nil {
				t.Fatal("ParsePgError returned nil")
			}
			if code := werror.Code(got); code != tt.wantCode {
				t.Fatalf("ParsePgError() code = %v, want %v", code, tt.wantCode)
			}
			if id := werror.ID(got); id != tt.wantID {
				t.Fatalf("ParsePgError() id = %q, want %q", id, tt.wantID)
			}
			if tt.wantIs != nil && !errors.Is(got, tt.wantIs) {
				t.Fatalf("ParsePgError() errors.Is(..., %v) = false", tt.wantIs)
			}
			for key, want := range tt.wantValues {
				gotValue, _ := werror.Value(got, key).(string)
				if gotValue != want {
					t.Fatalf("ParsePgError() value %s = %q, want %q", key, gotValue, want)
				}
			}
		})
	}

	if got := m.ParsePgError(pgx.ErrTxClosed); got != pgx.ErrTxClosed {
		t.Fatalf("non-pg sentinel must pass through, got %v", got)
	}

	// a second pass must not wrap the already mapped error again
	once := m.ParsePgError(uniqueErr())
	if m.ParsePgError(once) != once {
		t.Fatal("double mapping")
	}
}

func TestParsePgErrorKeepsCause(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "space_name_uq"}

	got := newErrorsManager().ParsePgError(pgErr)

	var cause *pgconn.PgError
	if !errors.As(got, &cause) || cause.ConstraintName != "space_name_uq" {
		t.Fatalf("ParsePgError() lost driver cause: %v", got)
	}
	if !errors.Is(got, ErrUnique) {
		t.Fatalf("ParsePgError() must match ErrUnique: %v", got)
	}
}

func TestParsePgErrorNotFoundKeepsErrNoRows(t *testing.T) {
	got := newErrorsManager().ParsePgError(pgx.ErrNoRows)
	if !errors.Is(got, ErrNotFound) {
		t.Fatalf("ParsePgError() must match ErrNotFound: %v", got)
	}
	if !errors.Is(got, pgx.ErrNoRows) {
		t.Fatalf("ParsePgError() must keep pgx.ErrNoRows cause: %v", got)
	}
	if got.Error() != ErrNotFound.Error() {
		t.Fatalf("ParsePgError() message = %q, want %q", got.Error(), ErrNotFound.Error())
	}
}

func TestParsePgErrorDataMessage(t *testing.T) {
	got := newErrorsManager().ParsePgError(&pgconn.PgError{Code: pgerrcode.InvalidTextRepresentation})
	if got.Error() != ErrData.Error() {
		t.Fatalf("ParsePgError() message = %q, want %q", got.Error(), ErrData.Error())
	}
}

func TestParsePgErrorPassesThroughCodedError(t *testing.T) {
	in := werror.New(
		"domain conflict",
		werror.WithCode(codes.AlreadyExists),
		werror.WithID("service.space.exists"),
	)

	got := newErrorsManager().ParsePgError(in)
	if got != in {
		t.Fatalf("ParsePgError() = %v, want same coded error", got)
	}
}

func TestErrUnreachableIsUnavailable(t *testing.T) {
	if code := werror.Code(ErrUnreachable); code != codes.Unavailable {
		t.Fatalf("ErrUnreachable code = %v, want %v", code, codes.Unavailable)
	}
	if id := werror.ID(ErrUnreachable); id != "pgw.unreachable" {
		t.Fatalf("ErrUnreachable id = %q, want %q", id, "pgw.unreachable")
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
	if err := (row{fakeRow{pgx.ErrNoRows}, p}).Scan(); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("row.Scan must keep ErrNoRows cause: %v", err)
	}
	if code := werror.Code((row{fakeRow{pgx.ErrNoRows}, p}).Scan()); code != codes.NotFound {
		t.Fatalf("row.Scan ErrNoRows code = %v, want NotFound", code)
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
