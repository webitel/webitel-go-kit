package etag

import (
	"strings"
	"unicode"
)

// FieldsCopy returns copy of unique set
// of given fields, all are in lower case
func FieldsCopy(fields []string) []string {
	// NOTE: in lower case
	return AddScope(nil, fields...)
}

// InlineFields explode inline 'attr,attr2 attr3' selector as ['attr','attr2','attr3']
func InlineFields(selector string) []string {
	// split func to explode inline userattrs selector
	split := func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}
	selector = strings.ToLower(selector)
	return strings.FieldsFunc(selector, split)
}

// SelectFields maps['*':userattrs, '+':userattrs+allattrs]
func SelectFields(userattrs, operational []string) func(string) []string {
	// split func to explode inline userattrs selector
	split := func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}
	return func(selector string) []string {
		selector = strings.ToLower(selector)
		fields := strings.FieldsFunc(selector, split)
		if len(fields) == 0 {
			return userattrs // imit '*'
		}
		for i := 0; i < len(fields); i++ {
			switch fields[i] {
			case "*":

				n := len(fields)
				fields = MergeFields(fields[:i],
					MergeFields(userattrs[:len(userattrs):len(userattrs)], fields[i+1:]))
				// advanced ?
				if len(fields) > n {
					i = len(fields) - n - 1
				}
			case "+":

				n := len(fields)
				fields = MergeFields(fields[:i], MergeFields(
					MergeFields(operational[:len(operational):len(operational)], userattrs),
					fields[i+1:],
				))
				// advanced ?
				if len(fields) > n {
					i = len(fields) - n - 1
				}
			}
		}
		return fields
	}
}

// FieldsFunc normalize a selection list src of the attributes to be returned.
//
//  1. An empty list with no attributes requests the return of all user attributes.
//  2. A list containing "*" (with zero or more attribute descriptions)
//     requests the return of all user attributes in addition to other listed (operational) attributes.
//
// e.g.: ['id,name','display'] returns ['id','name','display']
func FieldsFunc(src []string, fn func(string) []string) []string {
	if len(src) == 0 {
		return fn("")
	}

	var dst []string
	for i := 0; i < len(src); i++ {
		// explode single selection attr
		switch set := fn(src[i]); len(set) {
		case 0: // none
			src = append(src[:i], src[i+1:]...)
			i-- // process this i again
		case 1: // one
			if len(set[0]) == 0 {
				src = append(src[:i], src[i+1:]...)
				i--
			} else if dst == nil {
				src[i] = set[0]
			} else {
				dst = MergeFields(dst, set)
			}
		default: // many
			// NOTE: should rebuild output
			if dst == nil && i > 0 {
				// copy processed entries
				dst = make([]string, i, len(src)-1+len(set))
				copy(dst, src[:i])
			}
			dst = MergeFields(dst, set)
		}
	}
	if dst == nil {
		return src
	}
	return dst
}

// MergeFields appends unique set from src to dst.
func MergeFields(dst, src []string) []string {
	if len(src) == 0 {
		return dst
	}
	//
	if cap(dst)-len(dst) < len(src) {
		ext := make([]string, len(dst), len(dst)+len(src))
		copy(ext, dst)
		dst = ext
	}

next: // append unique set of src to dst
	for _, attr := range src {
		if len(attr) == 0 {
			continue
		}
		// look backwords for duplicates
		for j := len(dst) - 1; j >= 0; j-- {
			if strings.EqualFold(dst[j], attr) {
				continue next // duplicate found
			}
		}
		// append unique attr
		dst = append(dst, attr)
	}
	return dst
}
