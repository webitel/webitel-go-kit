package internal

import (
	"reflect"
	"testing"
)

func TestGetScheme(t *testing.T) {
	for _, tc := range []struct {
		dsn, scheme, rest string
		wantErr           bool
	}{
		{"stdout", "stdout", "", false},
		{"file:/var/log/m.json;max-size=10", "file", "/var/log/m.json;max-size=10", false},
		{"otlp+http://host", "otlp+http", "//host", false},
		{"/var/log/m.json", "", "/var/log/m.json", false},
		{"1file:x", "", "1file:x", false},
		{":x", "", "", true},
	} {
		scheme, rest, err := GetScheme(tc.dsn)
		if (err != nil) != tc.wantErr || scheme != tc.scheme || rest != tc.rest {
			t.Errorf("GetScheme(%q) = %q, %q, %v; want %q, %q, err=%v",
				tc.dsn, scheme, rest, err, tc.scheme, tc.rest, tc.wantErr)
		}
	}
}

func TestParseDSN(t *testing.T) {
	for _, tc := range []struct {
		dsn, path string
		params    map[string]string
		wantErr   bool
	}{
		{"file:/var/log/m.json", "/var/log/m.json", nil, false},
		{"file:/var/log/m.json;max-size=10;compress=gzip", "/var/log/m.json",
			map[string]string{"max-size": "10", "compress": "gzip"}, false},
		{"file:///var/log/m.json?max-size=10&backups=2#frag", "/var/log/m.json",
			map[string]string{"max-size": "10", "backups": "2"}, false},
		{"/var/log/m.json", "/var/log/m.json", nil, false},
		{":x", "", nil, true},
	} {
		path, params, err := ParseDSN(tc.dsn)
		if (err != nil) != tc.wantErr || path != tc.path || !reflect.DeepEqual(params, tc.params) {
			t.Errorf("ParseDSN(%q) = %q, %v, %v; want %q, %v, err=%v",
				tc.dsn, path, params, err, tc.path, tc.params, tc.wantErr)
		}
	}
}
