//go:generate stringer -type=EtagType
package etag

// EtagType represents the [E]Tag object type reference.
type EtagType uint32

// Case-related ETag types are declared here.
// **For generating string :  **
// go install golang.org/x/tools/cmd/stringer@latest
// echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc
// source ~/.zshrc
// go generate ./...
// **Commands**
const (
	NoType          EtagType = iota // NoType represents an unknown or invalid type
	EtagCase                        // Case type
	EtagCaseComment                 // Case Comment type
	EtagCaseLink                    // Case Link type
	EtagRelatedCase                 // Case Related case type
)

// **maxWellKnownType** is updated after the case types to track the highest value.
// Future blocks for other business entities will start counting from this value.
const maxWellKnownType = EtagCaseLink

// validType checks if the provided type is valid.
func validType(typ EtagType) bool {
	return NoType < typ
}

/*
**How to Add New Types for Other Business Entities:**

1. **Determine the Base Type:**
   - Each new block of types should start counting from the highest existing known value (tracked by `maxWellKnownType`).
   - Use the pattern `iota + maxWellKnownType` to define new types.
   - This ensures that new types for other business entities don't overlap with previously defined types.

2. **Naming Convention:**
   - New types should follow the pattern `Etag<ObjectName><ObjectAttributes>`.
   - Ensure that the type names start with `Etag` and then describe the object and attributes clearly.

3. **Updating the Maximum Known Type:**
   - After defining the new block of types, update the `maxWellKnownType` constant to track the highest value.
   - This ensures that future type blocks will start from the correct position.

**Example:**
If adding types for a new business entity like "Contact", the process would look like this:

```go
// Define the base for order-related ETag types.
const (
	_                          = iota + maxWellKnownType // Start from the last known highest value.
	EtagContact         EtagType = iota + maxWellKnownType // Contact type
	EtagContactEmail                                       // Contact Email type
	EtagContactLabel                                    // Contact Label type
)

// Update maxWellKnownType to reflect the new highest value.
const maxWellKnownType = EtagContactLabel
*/
