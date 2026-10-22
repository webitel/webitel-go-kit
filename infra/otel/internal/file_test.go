package internal

import "testing"

func TestFileDSN(t *testing.T) {
	type fields struct {
		name             string
		size, age, count int
		local, compress  bool
	}
	defaults := fields{"/var/log/m.json", 100, 30, 3, true, false}

	for _, tc := range []struct {
		dsn     string
		want    fields
		wantErr bool
	}{
		{"file:/var/log/m.json", defaults, false},
		{"file:/var/log/m.json;max-size=10;max-age=7;backups=1;localtime=utc;compress=gzip",
			fields{"/var/log/m.json", 10, 7, 1, false, true}, false},
		{"file:/var/log/m.json;max-size=ten", defaults, false}, // bad value: default kept
		{"file:/var/log/m.json;colour=red", defaults, false},   // unknown param: ignored
		{"file:", fields{}, true},
		{"file:/", fields{}, true},
	} {
		w, err := FileDSN(tc.dsn)
		if (err != nil) != tc.wantErr {
			t.Errorf("FileDSN(%q) err = %v, want err=%v", tc.dsn, err, tc.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		got := fields{w.Filename, w.MaxSize, w.MaxAge, w.MaxBackups, w.LocalTime, w.Compress}
		if got != tc.want {
			t.Errorf("FileDSN(%q) = %+v, want %+v", tc.dsn, got, tc.want)
		}
	}
}
