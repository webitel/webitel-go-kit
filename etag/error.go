package etag

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/webitel/webitel-go-kit/errors"
)

// Example input and output for the error message construction:

//   **Input:**
//    Type constant: `EtagCaseLink`

//    **Transformation:**
//    - The constant name "EtagCaseLink" is passed into the `extractTypeName` function.
//    - The "Etag" prefix is removed.
//    - The remaining string "CaseLink" is transformed to lowercase and split into dot notation: "case.link".

//    **Output:**
//    The final error message would be constructed as `"etag.case.link.<msg>"`.

// extractTypeName dynamically parses the name of the EtagType and converts it to dot notation.
func extractTypeName(typ EtagType) string {
	// Get the string representation of the constant name.
	typeName := typ.String()

	// If the constant starts with "Etag", remove the prefix.
	if strings.HasPrefix(typeName, "Etag") {
		// Remove the "Etag" prefix.
		trimmedName := typeName[4:]

		// Convert camelCase to dot notation using regex.
		re := regexp.MustCompile("([a-z0-9])([A-Z])")
		dotNotation := re.ReplaceAllString(trimmedName, "${1}.${2}")

		// Convert to lowercase for consistent naming convention.
		return strings.ToLower(dotNotation)
	}

	// Return "unknown" if it doesn't start with "Etag".
	return "unknown"
}

// Example usage for error message construction:

// **Example:**
//   **Input:** EtagCaseLink and message key `not_found`
//   **Output:** "etag.case.link.not_found"

// errorMessage returns the error message prefix based on the ETag type in dot notation.
func errorMessage(typ EtagType, msg string) string {
	typeName := extractTypeName(typ)
	return fmt.Sprintf("etag.%s.%s", typeName, msg) // Construct error message with dot notation
}

// **Example:**
//   **Input:** EtagCaseLink, `not_found`, and details like `"case link id: %s"`.
//   **Output:** An error message like `"etag.case.link.not_found"` with formatted details.

// NewBadRequestError generates a formatted bad request error based on the type and error message.
func NewBadRequestError(typ EtagType, msg string, details string, args ...interface{}) errors.AppError {
	return errors.NewBadRequestError(
		errorMessage(typ, msg),
		fmt.Sprintf(details, args...),
	)
}

// **Example:**
//   **Input:** EtagCase, `internal_error`, and details like `"case id: %s"`.
//   **Output:** An internal server error message like `"etag.case.internal_error"` with formatted details.

// NewInternalError generates a formatted internal server error based on the type and error message.
func NewInternalError(typ EtagType, msg string, details string, args ...interface{}) errors.AppError {
	return errors.NewInternalError(
		errorMessage(typ, msg),
		fmt.Sprintf(details, args...),
	)
}
