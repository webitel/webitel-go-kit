package stdout

import (
	"context"
	"testing"
)

func Test_withOptions(t *testing.T) {
	type args struct {
		ctx    context.Context
		rawDSN string
	}
	tests := []struct {
		name string
		args args
		// want    []log.Option
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "stdout:",
			wantErr: false,
		},
		{
			name:    "stderr://",
			wantErr: false,
		},
		{
			name:    "file",
			wantErr: true,
		},
		{
			name:    "file:",
			wantErr: true,
		},
		{
			name:    "file://",
			wantErr: true,
		},
		{
			name:    "file:///",
			wantErr: true,
		},
		{
			name:    "file:////////",
			wantErr: true,
		},
		{
			name:    "file:relative/path/file.log",
			wantErr: false,
		},
		{
			name:    "file://relative/path/file.log",
			wantErr: false,
		},
		{
			name:    "file:///absolute/path/file.log",
			wantErr: false,
		},
		{
			name:    "file:/absolute/path/file.log",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := withOptions(context.TODO(), tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("withOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("withOptions() = %v, want %v", got, tt.want)
			// }
		})
	}
}
