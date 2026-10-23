package health

import (
	"context"
	"testing"
)

func pass(context.Context) error { return nil }

func keys(results []CheckResult) []string {
	out := make([]string, len(results))
	for i, res := range results {
		out[i] = res.Key()
	}

	return out
}

func TestKey(t *testing.T) {
	cases := []struct {
		res  CheckResult
		want string
	}{
		{CheckResult{System: SystemPostgreSQL}, "postgresql"},
		{CheckResult{System: SystemPostgreSQL, Name: "standby"}, "postgresql/standby"},
	}

	for _, tc := range cases {
		if got := tc.res.Key(); got != tc.want {
			t.Errorf("Key() = %q, want %q", got, tc.want)
		}
	}
}

func TestSameSystemWithNames(t *testing.T) {
	reg := New(testConfig(), nil)
	reg.Informational(SystemPostgreSQL, pass, WithName("standby"))
	reg.Informational(SystemPostgreSQL, pass)

	results := reg.results()
	if got, want := keys(results), []string{"postgresql", "postgresql/standby"}; !equalStrings(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	if results[1].System != SystemPostgreSQL || results[1].Name != "standby" {
		t.Fatalf("named check = %+v, want system postgresql and name standby", results[1])
	}
	if results[0].Name != "" {
		t.Fatalf("unnamed check has name %q", results[0].Name)
	}
}

func TestEmptyNameIsNoName(t *testing.T) {
	reg := New(testConfig(), nil)
	reg.Informational(SystemPostgreSQL, pass, WithName(""))
	reg.Informational(SystemPostgreSQL, pass)

	results := reg.results()
	if got, want := keys(results), []string{"postgresql"}; !equalStrings(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	if results[0].Name != "" {
		t.Fatalf("name = %q, want empty", results[0].Name)
	}
}

func TestCustomSystem(t *testing.T) {
	reg := start(t, testConfig())
	reg.Critical("freeswitch", pass)

	waitFor(t, "freeswitch to pass", func() bool { return find(reg, "freeswitch").Status == StatusOK })

	if got := find(reg, "freeswitch").System; got != "freeswitch" {
		t.Fatalf("system = %q, want freeswitch", got)
	}
}

func TestNilOptionIsIgnored(t *testing.T) {
	reg := New(testConfig(), nil)
	reg.Informational(SystemPostgreSQL, pass, nil, WithName("standby"))

	if got, want := keys(reg.results()), []string{"postgresql/standby"}; !equalStrings(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
}
