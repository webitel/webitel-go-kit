package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetails(t *testing.T) {
	// nil -> empty
	assert.Empty(t, Details(nil))

	err := New("bang", WithValue("raw", "stay calm"), WithValue("color", "red"))
	assert.Equal(t, "bang", err.Error())
	assert.Equal(t, "bang; color = red; raw = stay calm", Details(err))
}
