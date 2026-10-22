package internal

import "testing"

func TestEnvString(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    string
		called bool
		want   string
	}{
		{"unset", "", false, ""},
		{"trimmed", "  stdout  ", true, "stdout"},
		{"unquoted", `"stdout"`, true, "stdout"},
		{"quoted empty", `""`, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &EnvReader{Namespace: "OTEL", GetEnv: func(key string) string {
				if key != "OTEL_METRICS_EXPORTER" {
					t.Fatalf("looked up %q", key)
				}
				return tc.env
			}}
			called, got := false, ""
			r.Apply(EnvString("METRICS_EXPORTER", func(s string) { called, got = true, s }))
			if called != tc.called || got != tc.want {
				t.Fatalf("called=%v value=%q, want called=%v value=%q", called, got, tc.called, tc.want)
			}
		})
	}
}

func TestGetEnvValueNamespace(t *testing.T) {
	env := map[string]string{"OTEL_X": "a", "X": "b"}
	get := func(key string) string { return env[key] }

	if v, _ := (&EnvReader{Namespace: "OTEL", GetEnv: get}).GetEnvValue("X"); v != "a" {
		t.Errorf("with namespace: %q, want a", v)
	}
	if v, _ := (&EnvReader{GetEnv: get}).GetEnvValue("X"); v != "b" {
		t.Errorf("without namespace: %q, want b", v)
	}
}
