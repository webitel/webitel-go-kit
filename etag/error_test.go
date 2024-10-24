package etag

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTypeName(t *testing.T) {
	assert.Equal(t, "case.comment", extractTypeName(EtagCaseComment), "Expected 'case.comment'")
	assert.Equal(t, "case.link", extractTypeName(EtagCaseLink), "Expected 'case.link'")
	assert.Equal(t, "unknown", extractTypeName(NoType), "Expected 'unknown' for NoType")
}

func TestErrorMessage(t *testing.T) {
	// Test constructing the error message for EtagCaseComment
	errMsg := errorMessage(EtagCaseComment, "not_found")
	assert.Equal(t, "etag.case.comment.not_found", errMsg, "Expected 'etag.case.comment.not_found'")

	// Test constructing the error message for an undefined type (it should return 'unknown')
	errMsgUnknown := errorMessage(NoType, "invalid")
	assert.Equal(t, "etag.unknown.invalid", errMsgUnknown, "Expected 'etag.unknown.invalid'")
}

func TestNewBadRequestError(t *testing.T) {
	// Test with a case link type
	err := NewBadRequestError(EtagCaseLink, "not_found", "case link id: 12345")
	assert.Equal(t, "etag.case.link.not_found", err.GetId(), "Error ID should be set correctly")
	assert.Equal(t, "case link id: 12345", err.GetDetailedError(), "Error details should be set correctly")

	// Test with a different type (such as EtagCaseComment)
	err = NewBadRequestError(EtagCaseComment, "invalid", "invalid case comment id: 67890")
	assert.Equal(t, "etag.case.comment.invalid", err.GetId(), "Error ID should be set correctly")
	assert.Equal(t, "invalid case comment id: 67890", err.GetDetailedError(), "Error details should be set correctly")

	// Testing a type that is not defined should return 'unknown'
	err = NewBadRequestError(NoType, "invalid", "unknown id: 000")
	assert.Equal(t, "etag.unknown.invalid", err.GetId(), "Expected 'etag.unknown.invalid'")
	assert.Equal(t, "unknown id: 000", err.GetDetailedError(), "Expected 'unknown id: 000'")
}

// TestNewInternalError tests the NewInternalError function.
func TestNewInternalError(t *testing.T) {
	// Test with a case link type
	err := NewInternalError(EtagCaseLink, "internal_error", "case link id: 12345")
	assert.Equal(t, "etag.case.link.internal_error", err.GetId(), "Error ID should be set correctly")
	assert.Equal(t, "case link id: 12345", err.GetDetailedError(), "Error details should be set correctly")
	assert.Equal(t, http.StatusInternalServerError, err.GetStatusCode(), "Status code should be 500 (Internal Server Error)")

	// Test with a case comment type
	err = NewInternalError(EtagCaseComment, "processing_failed", "case comment id: 67890")
	assert.Equal(t, "etag.case.comment.processing_failed", err.GetId(), "Error ID should be set correctly")
	assert.Equal(t, "case comment id: 67890", err.GetDetailedError(), "Error details should be set correctly")
	assert.Equal(t, http.StatusInternalServerError, err.GetStatusCode(), "Status code should be 500 (Internal Server Error)")

	// Test with a type that is not defined, which should fallback to "unknown"
	err = NewInternalError(NoType, "invalid", "unknown error occurred")
	assert.Equal(t, "etag.unknown.invalid", err.GetId(), "Error ID should be set correctly")
	assert.Equal(t, "unknown error occurred", err.GetDetailedError(), "Error details should be set correctly")
	assert.Equal(t, http.StatusInternalServerError, err.GetStatusCode(), "Status code should be 500 (Internal Server Error)")
}
