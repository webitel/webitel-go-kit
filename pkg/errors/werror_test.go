package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	werror "github.com/webitel/webitel-go-kit/pkg/errors"
)

func TestNew(t *testing.T) {
	err := werror.New("bang")
	assert.EqualError(t, err, "bang")

	// werror.New accepts werror.Wrapper options
	err = werror.New("boom", werror.WithValue("raw", "blue"))
	assert.EqualError(t, err, "boom")
	assert.Equal(t, "blue", werror.Value(err, "raw"))
}

func TestCodedConstructors(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string, ...werror.Wrapper) error
		code codes.Code
	}{
		{name: "Forbidden", fn: werror.Forbidden, code: codes.PermissionDenied},
		{name: "Unauthenticated", fn: werror.Unauthenticated, code: codes.Unauthenticated},
		{name: "NotFound", fn: werror.NotFound, code: codes.NotFound},
		{name: "AlreadyExists", fn: werror.AlreadyExists, code: codes.AlreadyExists},
		{name: "InvalidArgument", fn: werror.InvalidArgument, code: codes.InvalidArgument},
		{name: "FailedPrecondition", fn: werror.FailedPrecondition, code: codes.FailedPrecondition},
		{name: "Aborted", fn: werror.Aborted, code: codes.Aborted},
		{name: "Internal", fn: werror.Internal, code: codes.Internal},
		{name: "Unavailable", fn: werror.Unavailable, code: codes.Unavailable},
		{name: "Canceled", fn: werror.Canceled, code: codes.Canceled},
		{name: "DeadlineExceeded", fn: werror.DeadlineExceeded, code: codes.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn("boom", werror.WithID("test.id"))
			assert.EqualError(t, err, "boom")
			assert.Equal(t, tt.code, werror.Code(err))
			assert.Equal(t, "test.id", werror.ID(err))
		})
	}
}

func TestWrap(t *testing.T) {
	// capture a stack
	ogerr := errors.New("boom")
	err := werror.Wrap(ogerr)

	// werror.New error should werror.Wrap the old error
	assert.True(t, errors.Is(err, ogerr))

	// werror.Wrap accepts werror.Wrapper args
	err = werror.Wrap(err, werror.WithValue("raw", "hi"), werror.WithCode(6))
	assert.Equal(t, "hi", werror.Value(err, "raw"))
	assert.Equal(t, codes.Code(6), werror.Code(err))

	// werror.Wrapping nil -> nil
	assert.Nil(t, werror.Wrap(nil))
}

func TestAppend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Append(nil, "big"))

	// append message
	assert.EqualError(t, werror.Append(werror.New("blue"), "big"), "blue: big")

	// werror.Wrapper varargs
	err := werror.Append(werror.New("blue"), "big", werror.WithCode(3))
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "blue: big")
}

func TestAppendf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Appendf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, werror.Appendf(werror.New("blue"), "big %s", "red"), "blue: big red")

	// werror.Wrapper varargs
	err := werror.Appendf(werror.New("blue"), "big %s", werror.WithCode(3), "red")
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "blue: big red")
}

func TestPrepend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Prepend(nil, "big"))

	// append message
	assert.EqualError(t, werror.Prepend(werror.New("blue"), "big"), "big: blue")

	// werror.Wrapper varargs
	err := werror.Prepend(werror.New("blue"), "big", werror.WithCode(3))
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "big: blue")
}

func TestPrependf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Prependf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, werror.Prependf(werror.New("blue"), "big %s", "red"), "big red: blue")

	// werror.Wrapper varargs
	err := werror.Prependf(werror.New("blue"), "big %s", werror.WithCode(3), "red")
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "big red: blue")
}

func TestValue(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Value(nil, "color"))

	err := werror.New("bang")
	assert.Nil(t, werror.Value(err, "color"))

	err = werror.Wrap(err, werror.WithValue("color", "red"))
	assert.Equal(t, "red", werror.Value(err, "color"))

	// will traverse non-werror in the chain
	err = werror.New("bam", werror.WithValue("color", "red"))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("raw", "yikes"))
	assert.Equal(t, "red", werror.Value(err, "color"))

	// will not search the cause chain
	err = werror.New("whoops", werror.WithCause(werror.New("yikes", werror.WithValue("color", "red"))))
	assert.Nil(t, werror.Value(err, "color"))

	// if the current error and the cause both have a werror.Value for the
	// same key, the top errors werror.Value will always take precedence, even
	// if the cause was attached to the error after the werror.Value was.

	err = werror.New("boom", werror.WithValue("color", "red"))
	err = werror.Wrap(err, werror.WithCause(werror.New("io error", werror.WithValue("color", "blue"))))
	assert.Equal(t, "red", werror.Value(err, "color"))
}

func TestValues(t *testing.T) {
	assert.Empty(t, werror.Values(nil))
	assert.Empty(t, werror.Values(errors.New("boom")))

	// create an error chain with a few values attached, and a non-werror
	// in the middle.
	err := werror.New("boom", werror.WithValue("raw", "bam"), werror.WithCode(4))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("color", "red"))

	values := werror.Values(err)

	assert.Equal(t, map[interface{}]interface{}{
		"raw":             "bam",
		werror.ErrKeyCode: codes.Code(4),
		"color":           "red",
	}, values)
}

func BenchmarkValues(b *testing.B) {
	// create an error chain with a few values attached
	// and a non-werror in the middle
	err := werror.New("boom", werror.WithValue("raw", "bam"), werror.WithCode(4))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("color", "red"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		werror.Values(err)
	}
}

func TestCode(t *testing.T) {
	// nil -> OK
	assert.Equal(t, codes.OK, werror.Code(nil))

	// default to Internal
	assert.Equal(t, codes.Internal, werror.Code(errors.New("boom")))

	// set with werror.Wrapper
	assert.Equal(t, codes.NotFound, werror.Code(werror.New("boom", werror.WithCode(codes.NotFound))))

	// works when werror.Value is deep in stack
	err := werror.New("bam", werror.WithCode(codes.NotFound))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("raw", "yikes"))
	assert.Equal(t, codes.NotFound, werror.Code(err))
}

func TestCause(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Cause(nil))

	// no cause -> nil
	assert.Nil(t, werror.Cause(werror.New("boom")))

	// with cause
	root := errors.New("boom")
	err := werror.New("yikes", werror.WithCause(root))
	assert.EqualError(t, werror.Cause(err), "boom")

	// with nil cause, should be no-op
	err = werror.New("yikes", werror.WithCause(nil))
	assert.Nil(t, werror.Cause(err))
}
