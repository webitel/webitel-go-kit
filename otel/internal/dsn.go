package internal

import "errors"

// Maybe rawDSN is of the form scheme:opts.
// (Scheme must be [a-zA-Z][a-zA-Z0-9+.-]*)
// If so, return scheme, opts; else return rawDSN, "".
func GetScheme(rawDSN string) (scheme, opts string, err error) {
	for i := 0; i < len(rawDSN); i++ {
		c := rawDSN[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				// invalid scheme spec !
				return "", rawDSN, nil
			}
		case c == ':':
			if i == 0 {
				return "", "", errors.New("missing scheme")
			}
			return rawDSN[:i], rawDSN[i+1:], nil
		default:
			// we have encountered an invalid character,
			// so there is no valid scheme
			return "", rawDSN, nil
		}
	}
	// return "", rawDSN, nil
	return rawDSN, "", nil
}
