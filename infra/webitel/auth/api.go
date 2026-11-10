package auth

import (
	"encoding/json"
	"fmt"
	"strings"
)

// service mirrors the WebitelServices type generated in each service package;
// go-kit cannot import it, so fields are matched by name.
type service struct {
	ObjClass           string
	AdditionalLicenses []string
	WebitelMethods     map[string]struct{ Access int }
}

// rules is the generated WebitelAPI map keyed by short service name.
type rules map[string]service

// rule is what a single gRPC method requires from the caller.
type rule struct {
	class    string
	licenses []string
	access   Action
}

func parseRules(webitelAPI any) (rules, error) {
	raw, err := json.Marshal(webitelAPI)
	if err != nil {
		return nil, fmt.Errorf("auth: encode WebitelAPI: %w", err)
	}

	var r rules
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("auth: WebitelAPI is not a map of services: %w", err)
	}
	if len(r) == 0 {
		return nil, fmt.Errorf("auth: WebitelAPI has no services")
	}
	// A renamed generated field decodes to zero values without an error.
	if !r.authorizable() {
		return nil, fmt.Errorf("auth: WebitelAPI has no method with an object class and a valid access, check the generated field names ObjClass, WebitelMethods and Access")
	}

	return r, nil
}

func (r rules) authorizable() bool {
	for _, svc := range r {
		if svc.ObjClass == "" {
			continue
		}
		for _, m := range svc.WebitelMethods {
			if m.Access >= int(ActionCreate) && m.Access <= int(ActionDelete) {
				return true
			}
		}
	}

	return false
}

// lookup resolves a full method name such as "/webitel.audit.Forms/Create". The
// generated map is keyed without the proto package, so only the service name
// after the last dot is matched.
func (r rules) lookup(fullMethod string) (rule, bool) {
	name, method, ok := strings.Cut(strings.TrimPrefix(fullMethod, "/"), "/")
	if !ok {
		return rule{}, false
	}
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name = name[i+1:]
	}

	svc, ok := r[name]
	if !ok || svc.ObjClass == "" {
		return rule{}, false
	}
	m, ok := svc.WebitelMethods[method]
	if !ok || m.Access < int(ActionCreate) || m.Access > int(ActionDelete) {
		return rule{}, false
	}

	return rule{class: svc.ObjClass, licenses: svc.AdditionalLicenses, access: Action(m.Access)}, true
}
