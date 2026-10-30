# pgw/apperr

Maps PostgreSQL and `pgw` errors to coded [`pkg/errors`](https://github.com/webitel/webitel-go-kit/tree/main/pkg/errors).

Separate module so `infra/pgw` does not depend on gRPC or application errors.

## Installation

```
go get github.com/webitel/webitel-go-kit/infra/pgw/apperr
```

## Usage

```go
import "github.com/webitel/webitel-go-kit/infra/pgw/apperr"

if err != nil {
    return apperr.ParseError(err)
}
```

| Input | gRPC code | ID |
|---|---|---|
| `pgx.ErrNoRows` | NotFound | `pgw.not_found` |
| `pgerrcode.UniqueViolation` | AlreadyExists | `pgw.unique` |
| `pgerrcode.ForeignKeyViolation` | FailedPrecondition | `pgw.foreign_key` |
| `pgerrcode.CheckViolation` | InvalidArgument | `pgw.check` |
| `pgerrcode.NotNullViolation` | InvalidArgument | `pgw.not_null` |
| `pgerrcode.SerializationFailure` / `DeadlockDetected` | Aborted | `pgw.aborted` |
| `pgerrcode.ProgramLimitExceeded` | InvalidArgument | `pgw.program_limit` |
| `*pgw.DataExceptionError` (no underlying `PgError`) | InvalidArgument | `pgw.data` |
| Anything else | Internal | `pgw.internal` |

Metadata from structured `pgconn` fields only — `Detail` is not parsed. Already-coded errors (e.g. from a constraint processor) pass through unchanged.
