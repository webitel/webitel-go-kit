package internal

import (
	"reflect"
	"testing"
)

func TestParseDSN(t *testing.T) {
	type args struct {
		dsn string
	}
	tests := []struct {
		name       string
		args       args
		wantPath   string
		wantParams map[string]string
		wantErr    bool
	}{
		// TODO: Add test cases.
		{
			name: "general",
			args: args{
				// dsn: "file:~/path/to/file.ext;size=100;age=7;backups=3;compress=gzip",
				// dsn: "file:~/path/to/file.ext?size=100&age=7&backups=3&compress=gzip",
				dsn: "file://~/path/to/file.ext?size=100&age=7&backups=3&compress=gzip#fragment",
				// dsn: "file://relative/path/to/file.ext?size=100&age=7&backups=3&compress=gzip",
				// dsn: "file:///absolute/path/to/file.ext?size=100&age=7&backups=3&compress=gzip",
			},
			wantPath: "~/path/to/file.ext",
			wantParams: map[string]string{
				"size":     "100",
				"age":      "7",
				"backups":  "3",
				"compress": "gzip",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotParams, err := ParseDSN(tt.args.dsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDSN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPath != tt.wantPath {
				t.Errorf("ParseDSN() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
			if !reflect.DeepEqual(gotParams, tt.wantParams) {
				t.Errorf("ParseDSN() gotParams = %v, want %v", gotParams, tt.wantParams)
			}
		})
	}
}
