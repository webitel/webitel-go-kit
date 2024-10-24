package etag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MockNode is a mock implementation of the INode interface for testing purposes.
type MockNode struct {
	ID string
}

func (n MockNode) GetId() string {
	return n.ID
}

func (n MockNode) ProtoReflect() protoreflect.Message {
	// Not used in this test case, so we return nil
	return nil
}

// TestRequireId tests the RequireId constraint function.
func TestRequireId(t *testing.T) {
	// Test with a valid node that has an ID.
	node := MockNode{ID: "12345"}
	validator := RequireId[MockNode]("test_node", EtagCase) // Explicitly specify MockNode as the type parameter
	err := validator(node)
	assert.NoError(t, err, "Expected no error for node with ID")

	// Test with an invalid node that has an empty ID.
	invalidNode := MockNode{ID: ""}
	validator = RequireId[MockNode]("test_node", EtagCaseComment) // Explicitly specify MockNode as the type parameter
	err = validator(invalidNode)
	assert.Error(t, err, "Expected an error for node with empty ID")

	// Since we're using AppError, we need to check if err is an AppError
	appErr, ok := err.(interface {
		GetId() string
		GetDetailedError() string
	})
	assert.True(t, ok, "Expected an AppError type")

	// Check error ID and detailed message
	assert.Equal(t, "etag.case.comment.id_missing", appErr.GetId(), "Expected error ID to be 'etag.case.comment.id_missing'")
	assert.Equal(t, "test_node (id: ); ID is required", appErr.GetDetailedError(), "Expected detailed error message")
}

// TestUniqueId tests the UniqueId function to check for ID conflicts.
func TestUniqueId(t *testing.T) {
	// Test with two nodes that have the same ID.
	node1 := MockNode{ID: "12345"}
	node2 := MockNode{ID: "12345"}
	err := UniqueId(node1, node2)
	assert.NoError(t, err, "Expected no error for two nodes with the same ID")

	// Test with two nodes that have different IDs.
	node1 = MockNode{ID: "12345"}
	node2 = MockNode{ID: "67890"}
	err = UniqueId(node1, node2)
	assert.NoError(t, err, "Expected no error for two nodes with different IDs")
}
