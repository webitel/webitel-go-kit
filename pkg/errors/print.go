package errors

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Details returns err.Error() and any additional details.
// The details of each error cause chain will also be printed.
// String keys from Values are emitted in sorted order for stable output.
func Details(err error) string {
	if err == nil {
		return ""
	}

	var b strings.Builder

	if id := ID(err); id != "" {
		fmt.Fprintf(&b, "%s: ", id)
	}

	b.WriteString(err.Error())

	values := Values(err)
	keys := make([]string, 0, len(values))

	for key := range values {
		if k, ok := key.(string); ok {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	for _, key := range keys {
		fmt.Fprintf(&b, "; %s = %v", key, values[key])
	}

	if cause := Cause(err); cause != nil {
		fmt.Fprintf(&b, "; caused by: %s", Details(cause))
	}

	return b.String()
}

// Format adapts errors to fmt.Formatter interface.
// It's intended to be used help error impls implement fmt.Formatter, e.g.:
//
//	    func (e *myErr) Format(f fmt.State, verb rune) {
//		     Format(f, verb, e)
//	    }
func Format(s fmt.State, verb rune, err error) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, Details(err)) //nolint:errcheck

			return
		}

		fallthrough
	case 's':
		io.WriteString(s, msgWithCauses(err)) //nolint:errcheck
	case 'q':
		fmt.Fprintf(s, "%q", err.Error())
	}
}

func msgWithCauses(err error) string {
	messages := make([]string, 0, 5)
	for err != nil {
		if ce := err.Error(); ce != "" {
			messages = append(messages, ce)
		}

		err = Cause(err)
	}

	return strings.Join(messages, ": ")
}
