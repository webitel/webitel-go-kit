package local

import (
	"net"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func Test_hashable(t *testing.T) {

	var (
		ip   = net.ParseIP("10.9.8.7").To4()
		arr  = [4]byte{0x0a, 0x09, 0x08, 0x07}
		uuid = uuid.UUID(uuid.New())
		hash = map[any]bool{}
	)

	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want any
	}{
		// TODO: Add test cases.
		{
			name: "net.IP",
			args: args{
				ip,
			},
			want: arr,
		},
		{
			name: "*net.IP",
			args: args{
				&ip,
			},
			want: arr,
		},
		{
			name: "array",
			args: args{
				arr,
			},
			want: arr,
		},
		{
			name: "*array",
			args: args{
				&arr,
			},
			want: arr,
		},
		{
			name: "UUID",
			args: args{
				uuid,
			},
			want: [16]byte(uuid),
		},
		{
			name: "*UUID",
			args: args{
				&uuid,
			},
			want: [16]byte(uuid),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hashable(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("hashable() = %#v, want %#v", got, tt.want)
			} else {
				hash[got] = true
			}
		})
	}
}
