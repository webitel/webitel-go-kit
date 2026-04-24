package interceptors

import (
	"context"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc"
)

func Test_logAndReturnGRPCError(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		err     error
		info    *grpc.UnaryServerInfo
		wantErr bool
	}{
		{
			name: "error without cause",
			err:  errors.New("test error"),
			info: &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
			wantErr: true,
		},
		{
			name: "error with cause",
			err: errors.InvalidArgument("invalid argument", errors.WithCause(errors.New("caused error"))),
			info: &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := logAndReturnGRPCError(context.Background(), tt.err, tt.info)
			if gotErr != nil && !tt.wantErr {
				t.Errorf("logAndReturnGRPCError() error = %v, wantErr %v", gotErr, tt.wantErr)
			}
		})
	}
}
