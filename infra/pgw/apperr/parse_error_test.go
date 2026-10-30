package apperr

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

func TestParseError(t *testing.T) {
	tests := []struct {
		name       string
		in         error
		wantCode   codes.Code
		wantID     string
		wantValues map[string]string
	}{
		{name: "nil passes through", wantCode: codes.OK},
		{
			name:     "no rows is not found",
			in:       pgx.ErrNoRows,
			wantCode: codes.NotFound,
			wantID:   "pgw.not_found",
		},
		{
			name:     "wrapped no rows is not found",
			in:       fmt.Errorf("scan: %w", pgx.ErrNoRows),
			wantCode: codes.NotFound,
			wantID:   "pgw.not_found",
		},
		{
			name:     "unique violation",
			in:       &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "space_name_uq"},
			wantCode: codes.AlreadyExists,
			wantID:   "pgw.unique",
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
			wantValues: map[string]string{
				"table":  "space",
				"column": "name",
			},
		},
		{
			name: "data exception without pg cause",
			in: &pgw.DataExceptionError{
				Schema: "ai_hub",
				Table:  "space",
				Column: "name",
				Detail: "null value",
			},
			wantCode: codes.InvalidArgument,
			wantID:   "pgw.data",
			wantValues: map[string]string{
				"schema": "ai_hub",
				"table":  "space",
				"column": "name",
			},
		},
		{
			name:     "connection exception without pg cause is internal",
			in:       &pgw.ConnectionExceptionError{Code: "08006", Detail: "connection failure"},
			wantCode: codes.Internal,
			wantID:   "pgw.internal",
		},
		{
			name:     "serialization failure is aborted",
			in:       &pgconn.PgError{Code: pgerrcode.SerializationFailure},
			wantCode: codes.Aborted,
			wantID:   "pgw.aborted",
		},
		{
			name:     "deadlock is aborted",
			in:       &pgconn.PgError{Code: pgerrcode.DeadlockDetected},
			wantCode: codes.Aborted,
			wantID:   "pgw.aborted",
		},
		{
			name:     "program limit exceeded is invalid argument",
			in:       &pgconn.PgError{Code: pgerrcode.ProgramLimitExceeded},
			wantCode: codes.InvalidArgument,
			wantID:   "pgw.program_limit",
		},
		{
			name:     "unknown pg error is internal",
			in:       &pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			wantCode: codes.Internal,
			wantID:   "pgw.internal",
		},
		{
			name:     "non-pg error is internal",
			in:       stderrors.New("connection reset"),
			wantCode: codes.Internal,
			wantID:   "pgw.internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseError(tt.in)
			if tt.in == nil {
				if got != nil {
					t.Fatalf("ParseError(nil) = %v, want nil", got)
				}

				return
			}

			if got == nil {
				t.Fatal("ParseError returned nil")
			}

			if code := errors.Code(got); code != tt.wantCode {
				t.Fatalf("ParseError() code = %v, want %v", code, tt.wantCode)
			}

			if id := errors.ID(got); id != tt.wantID {
				t.Fatalf("ParseError() id = %q, want %q", id, tt.wantID)
			}

			for key, want := range tt.wantValues {
				gotValue, _ := errors.Value(got, key).(string)
				if gotValue != want {
					t.Fatalf("ParseError() value %s = %q, want %q", key, gotValue, want)
				}
			}
		})
	}
}

func TestParseErrorKeepsCause(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "space_name_uq"}

	got := ParseError(pgErr)

	var cause *pgconn.PgError
	if !stderrors.As(got, &cause) || cause.ConstraintName != "space_name_uq" {
		t.Fatalf("ParseError() lost driver cause: %v", got)
	}
}

func TestParseErrorKeepsDataExceptionCause(t *testing.T) {
	in := &pgw.DataExceptionError{Schema: "ai_hub", Table: "space", Column: "name"}

	got := ParseError(in)

	var cause *pgw.DataExceptionError
	if !stderrors.As(got, &cause) || cause.Table != "space" {
		t.Fatalf("ParseError() lost data exception cause: %v", got)
	}
}

func TestParseErrorPassesThroughCodedError(t *testing.T) {
	in := errors.New(
		"domain conflict",
		errors.WithCode(codes.AlreadyExists),
		errors.WithID("service.space.exists"),
	)

	got := ParseError(in)
	if got != in {
		t.Fatalf("ParseError() = %v, want same coded error", got)
	}
}
