package pgw

import (
	"errors"

	werror "github.com/webitel/webitel-go-kit/pkg/errors"
)

var (
	ErrUnreachable = werror.Unavailable(
		"the resource is unreachable now",
		werror.WithID("pgw.unreachable"),
	)
	ErrClosed          = errors.New("pgw: pool manager is closed")
	ErrStarted         = errors.New("pgw: pool manager is already started")
	ErrMigrationVerify = errors.New("pgw: migration unverified")

	ErrNotFound = werror.NotFound(
		"entity does not exist or access is denied",
		werror.WithID("pgw.not_found"),
	)
	ErrUnique = werror.AlreadyExists(
		"invalid input: entity already exists",
		werror.WithID("pgw.unique"),
	)
	ErrForeignKey = werror.FailedPrecondition(
		"invalid input: referenced entity does not exist or is still referenced",
		werror.WithID("pgw.foreign_key"),
	)
	ErrCheck = werror.InvalidArgument(
		"invalid input: value violates a constraint",
		werror.WithID("pgw.check"),
	)
	ErrNotNull = werror.InvalidArgument(
		"invalid input: required value is missing",
		werror.WithID("pgw.not_null"),
	)
	ErrData = werror.InvalidArgument(
		"invalid input: malformed or out of range value",
		werror.WithID("pgw.data"),
	)
	ErrConnection = werror.Unavailable(
		"database connection error",
		werror.WithID("pgw.connection"),
	)
	ErrAborted = werror.Aborted(
		"transaction conflict; retry the operation",
		werror.WithID("pgw.aborted"),
	)
	ErrProgramLimit = werror.InvalidArgument(
		"invalid input: program limit exceeded",
		werror.WithID("pgw.program_limit"),
	)
	ErrCanceled = werror.Canceled(
		"operation canceled",
		werror.WithID("pgw.canceled"),
	)
	ErrDeadlineExceeded = werror.DeadlineExceeded(
		"operation deadline exceeded",
		werror.WithID("pgw.deadline_exceeded"),
	)
	ErrInternal = werror.Internal(
		"storage error",
		werror.WithID("pgw.internal"),
	)
)
