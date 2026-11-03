package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	genServicesInfo map[string]genServices
	genServices     struct {
		ObjClass           string
		AdditionalLicenses []string
		WebitelMethods     map[string]genMethod
	}
	genMethod struct {
		HttpBindings []*genBinding
		Access       int
		Input        string
		Output       string
	}
	genBinding struct{ Path, Method string }
)

var testAPI = genServicesInfo{
	"Forms": {
		ObjClass:           "forms",
		AdditionalLicenses: []string{"AUDIT"},
		WebitelMethods: map[string]genMethod{
			"Create": {Access: 0, Input: "CreateRequest", HttpBindings: []*genBinding{{Path: "/forms", Method: "post"}}},
			"Search": {Access: 1},
			"Update": {Access: 2},
			"Delete": {Access: 3},
			"Bad":    {Access: 7},
		},
	},
	"NoClass": {WebitelMethods: map[string]genMethod{"Do": {Access: 1}}},
}

func TestParseRules(t *testing.T) {
	r, err := parseRules(testAPI)
	require.NoError(t, err)
	assert.Len(t, r, 2)

	_, err = parseRules("not a map")
	assert.Error(t, err)

	_, err = parseRules(genServicesInfo{})
	assert.Error(t, err)

	_, err = parseRules(nil)
	assert.Error(t, err)

	// A renamed field decodes to zero values: the map is non-empty but unusable.
	type renamed struct {
		Class   string
		Methods map[string]struct{ Access int }
	}
	_, err = parseRules(map[string]renamed{"Forms": {Class: "forms", Methods: map[string]struct{ Access int }{"Search": {Access: 1}}}})
	assert.ErrorContains(t, err, "ObjClass")
}

func TestLookup(t *testing.T) {
	r, err := parseRules(testAPI)
	require.NoError(t, err)

	tests := []struct {
		method string
		want   rule
		ok     bool
	}{
		{"/webitel.audit.Forms/Create", rule{"forms", []string{"AUDIT"}, Create}, true},
		{"/webitel.audit.Forms/Search", rule{"forms", []string{"AUDIT"}, Read}, true},
		{"/Forms/Update", rule{"forms", []string{"AUDIT"}, Update}, true},
		{"/webitel.audit.Forms/Delete", rule{"forms", []string{"AUDIT"}, Delete}, true},
		{"/webitel.audit.Forms/Bad", rule{}, false},
		{"/webitel.audit.Forms/Missing", rule{}, false},
		{"/webitel.audit.Unknown/Create", rule{}, false},
		{"/webitel.audit.NoClass/Do", rule{}, false},
		{"garbage", rule{}, false},
		{"", rule{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got, ok := r.lookup(tt.method)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
