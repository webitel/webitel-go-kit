// Package contacts provides ETag encoding/decoding for contact-related
// objects, wire-compatible with the contacts service identifiers.
package contacts

// Type represents [E]Tag object type reference.
type Type uint8

// ETag reference object types. Numeric values are frozen: they are
// encoded into issued etag strings and must never be renumbered.
const (
	// NoType represents an unknown or invalid type.
	NoType Type = iota

	SrcType      // Contact
	NameType     // Contact Name
	LabelType    // Contact Label
	EmailType    // Contact Email
	PhoneType    // Contact Phone
	GroupType    // Contact Group
	AddressType  // Contact Address
	ManagerType  // Contact Manager
	CommentType  // Contact Comment
	VariableType // Contact Variable
	LanguageType // Contact Language
	TimezoneType // Contact Timezone
	IMClientType // Contact IM Client

	SpaceType   // Space
	ArticleType // Article
)

// validType checks if the provided type is valid.
func validType(typ Type) bool {
	return NoType < typ
}
