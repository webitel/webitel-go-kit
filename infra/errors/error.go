package errors

import (
	"cmp"
	"fmt"
	"net/http"
	"strings"

	pbrpc "github.com/webitel/protos/gen/go/rpc"

	pbstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Parse tries to parse a JSON string into an error. If that
// fails, it will set the given string as the error detail.
func Parse(message string) (dst *Error, ok bool) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, true
	}
	src := new(pbrpc.Error)
	enc := codecPlain
	err := enc.Unmarshal(
		[]byte(message), src,
	)
	if err != nil {
		// original [message] text
		src.Message = message
	}
	return FromProto(src), (err == nil)
}

// FromProto type conversion method
func FromProto(src *pbrpc.Error) *Error {
	return (*Error)(src)
}

// FromError type conversion method
func FromError(src error) (dst *Error, ok bool) {
	if src == nil {
		return nil, true
	}
	switch src := src.(type) {
	case *Error:
		{
			return src, true
		}
	}
	type grpcstatus interface {
		GRPCStatus() *status.Status
	}
	if impl, ok := src.(grpcstatus); ok {
		return FromStatus(impl.GRPCStatus())
	}
	return Parse(src.Error())
}

// FromStatus type conversion method
func FromStatus(src *status.Status) (dst *Error, ok bool) {
	if src == nil {
		return nil, true
	}
	for _, any := range src.Proto().GetDetails() {
		sub, err := any.UnmarshalNew()
		if err != nil {
			// details = append(details, err)
			continue
		}
		switch e := sub.(type) {
		case *pbrpc.Error:
			{
				return (*Error)(e), true
			}
		}
	}

	// [finally]: try to parse JSON string
	if dst, ok = Parse(src.Message()); !ok {
		dst.Code = int32(src.Code())
		dst.Status = src.Code().String()
	}

	return // err, ok?
}

// An internal Error details
type Error pbrpc.Error

// func (err *Error) Code() int32 {}
// func (err *Error) Status() string {}
// func (err *Error) Message() string {}

// Proto returns [e] as an *rpcpb.Error proto message.
func (e *Error) proto() *pbrpc.Error {
	return (*pbrpc.Error)(e)
}

// Proto returns [e] as an rpcpb.Error proto message.
func (e *Error) Proto() *pbrpc.Error {
	if e == nil {
		return nil
	}
	return proto.CloneOf(e.proto())
}

// ProtoAny encodes [e] into [protobuf.Any] message structure
func (e *Error) ProtoAny() (*anypb.Any, error) {
	// if e == nil {} // This is error !
	return anypb.New(e.proto())
}

var _ error = (*Error)(nil)

var codecPlain = struct {
	protojson.MarshalOptions
	protojson.UnmarshalOptions
}{
	MarshalOptions: protojson.MarshalOptions{
		Multiline:         false,
		Indent:            "",
		AllowPartial:      false,
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		EmitUnpopulated:   false,
		EmitDefaultValues: false,
		Resolver:          nil,
	},
	UnmarshalOptions: protojson.UnmarshalOptions{
		AllowPartial:   false,
		DiscardUnknown: false,
		RecursionLimit: 0,
		Resolver:       nil,
	},
}

// Error implements Go error interface
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return codecPlain.Format(e.proto())
}

func (e *Error) String() string {

	if e == nil {
		return ""
	}

	var (
		indent string
		format strings.Builder
	)
	defer format.Reset()

	if e.Code > 0 {
		// format.WriteString(indent)
		fmt.Fprintf(&format, "(#%d)", e.Code)
		indent = " "
	}

	if e.Status != "" {
		format.WriteString(indent)
		format.WriteString(e.Status)
		indent = " ; "
	}

	if e.Message != "" {
		format.WriteString(indent)
		format.WriteString(e.Message)
	}

	return format.String()
}

// GRPCStatus returns the [grpc.Status] represented by [e].
// Compatibility for grpc/status.FromError() method.
func (e *Error) GRPCStatus() *status.Status {
	src := e.proto()
	top := &pbstatus.Status{
		Code:    int32(http2grpcCode(src.GetCode())),
		Message: cmp.Or(src.GetStatus(), src.GetMessage()),
		// Details: []*anypb.Any{sub},
	}
	sub, re := e.ProtoAny()
	if re != nil {
		top.Message = e.Error() // JSON
		// sub, _ = anypb.New(&rpcpb.Error{
		// 	// Id:      "",
		// 	Code:    500,
		// 	Status:  "Server Internal Error",
		// 	Message: re.Error(),
		// })
	} else {
		top.Details = []*anypb.Any{sub}
	}
	// top := &spb.Status{
	// 	Code:    int32(http2grpcCode(src.GetCode())),
	// 	Message: cmp.Or(src.GetStatus(), src.GetMessage()),
	// 	Details: []*anypb.Any{sub},
	// }
	return status.FromProto(top)
}

// Option to setup Error details
type Option func(err *Error)

// Error.Code number Option
func Code(code int32) Option {
	return func(err *Error) {
		if code > 0 {
			err.Code = code
		}
	}
}

// Error.Status code Option
func Status(code string) Option {
	return func(err *Error) {
		if code != "" {
			err.Status = code
		}
	}
}

// Error.Message format Option
func Message(form string, args ...any) Option {
	return func(err *Error) {
		text := form
		if len(args) > 0 {
			if form == "" {
				text = fmt.Sprint(args...)
			} else {
				text = fmt.Sprintf(form, args...)
			}
		}
		err.Message = text
	}
}

// New Error with Options..
func New(opts ...Option) (err *Error) {
	err = &Error{}
	err.init(opts)
	return // err
}

func (e *Error) init(opts []Option) {
	for _, setup := range opts {
		setup(e)
	}
}

// Errorf
func Errorf(message string, args ...any) *Error {
	return New(Message(message, args...))
}

// (#400) BAD_REQUEST
//
//	 New(
//		Status("BAD_REQUEST"),
//		Code(http.StatusBadRequest),
//		opts...,
//	)
func BadRequest(opts ...Option) *Error {
	err := New(
		Status("BAD_REQUEST"),
		Code(http.StatusBadRequest),
	)
	err.init(opts)
	return err
}

// (#401) UNAUTHORIZED
//
//	 New(
//		Status("UNAUTHORIZED"),
//		Code(http.StatusUnauthorized),
//		opts...,
//	)
func Unauthorized(opts ...Option) *Error {
	err := New(
		Status("UNAUTHORIZED"),
		Code(http.StatusUnauthorized),
	)
	err.init(opts)
	return err
}

// (#403) FORBIDDEN
//
//	 New(
//		Status("FORBIDDEN"),
//		Code(http.StatusForbidden),
//		opts...,
//	)
func Forbidden(opts ...Option) *Error {
	err := New(
		Status("FORBIDDEN"),
		Code(http.StatusForbidden),
	)
	err.init(opts)
	return err
}

// (#404) NOT_FOUND
//
//	 New(
//		Status("NOT_FOUND"),
//		Code(http.StatusNotFound),
//		opts...,
//	)
func NotFound(opts ...Option) *Error {
	err := New(
		Status("NOT_FOUND"),
		Code(http.StatusNotFound),
	)
	err.init(opts)
	return err
}

// (#500) INTERNAL
//
//	 New(
//		Status("INTERNAL"),
//		Code(http.StatusInternalServerError),
//		opts...,
//	)
func Internal(opts ...Option) *Error {
	err := New(
		Status("INTERNAL"),
		Code(http.StatusInternalServerError),
	)
	err.init(opts)
	return err
}

// (#502) BAD_GATEWAY
//
//	 New(
//		Status("BAD_GATEWAY"),
//		Code(http.StatusBadGateway),
//		opts...,
//	)
func BadGateway(opts ...Option) *Error {
	err := New(
		Status("BAD_GATEWAY"),
		Code(http.StatusBadGateway),
	)
	err.init(opts)
	return err
}
