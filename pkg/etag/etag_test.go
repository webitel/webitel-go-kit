package etag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestValidType tests the validType function
func TestValidType(t *testing.T) {
	assert.True(t, validType(EtagCase), "EtagCase should be a valid type")
	t.Log("EtagCase is a valid type")

	assert.False(t, validType(NoType), "NoType should not be a valid type")
	t.Log("NoType is an invalid type")
}

// TestTid_GetOid tests the GetOid function of Tid
func TestTid_GetOid(t *testing.T) {
	tid := Tid{Oid: 12345}
	assert.Equal(t, int64(12345), tid.GetOid(), "Expected OID to be 12345")
	t.Logf("Tid.GetOid returned expected value: %d", tid.GetOid())

	tidZero := Tid{}
	assert.Equal(t, int64(0), tidZero.GetOid(), "Expected OID to be 0 for an empty Tid")
	t.Logf("Tid.GetOid returned expected value for empty Tid: %d", tidZero.GetOid())
}

// TestTid_Valid tests the Valid method of Tid for different scenarios
func TestTid_Valid(t *testing.T) {
	tid := Tid{Oid: 12345}
	err := tid.Valid()
	assert.NoError(t, err, "Tid with OID 12345 should be valid")
	t.Log("Tid with OID 12345 is valid")

	tidInvalid := Tid{Oid: 0}
	err = tidInvalid.Valid()
	assert.Error(t, err, "Tid with OID 0 should be invalid")
	t.Log("Tid with OID 0 is invalid as expected")
}

// TestETag_Valid tests the Valid method of ETag
func TestETag_Valid(t *testing.T) {
	etag := ETag{Tid: Tid{Oid: 12345}, EtagType: EtagCase}
	err := etag.Valid()
	assert.NoError(t, err, "ETag with valid Tid and EtagType should be valid")
	t.Log("ETag with valid Tid and EtagType is valid")

	etagInvalid := ETag{Tid: Tid{Oid: 0}, EtagType: EtagCase}
	err = etagInvalid.Valid()
	assert.Error(t, err, "ETag with invalid Tid should be invalid")
	t.Log("ETag with invalid Tid is invalid as expected")

	etagMissingType := ETag{Tid: Tid{Oid: 12345}, EtagType: NoType}
	err = etagMissingType.Valid()
	assert.Error(t, err, "ETag with missing type should be invalid")
	t.Log("ETag with missing type is invalid as expected")
}

// TestEncodeEtag tests the EncodeEtag function
func TestEncodeEtag(t *testing.T) {
	encoded, err := EncodeEtag(EtagCase, 12345, 0)
	if err != nil {
		t.Error(err.Error())
	}
	assert.NotEmpty(t, encoded, "Encoded ETag should not be empty")
	t.Logf("Successfully encoded ETag: %s", encoded)
	_, err = EncodeEtag(NoType, 12345, 0)
	assert.Error(t, err, "Encoding with NoType should return error")

	_, err = EncodeEtag(EtagCase, 0, 0)

	assert.Error(t, err, "Encoding with OID 0 should panic")
}

// TestDecodeEtag tests the DecodeEtag function
func TestDecodeEtag(t *testing.T) {
	encoded, _ := EncodeEtag(EtagCase, 12345, 0)
	typ, oid, ver, err := DecodeEtag(encoded)
	assert.NoError(t, err, "Decoding valid ETag should not produce an error")
	assert.Equal(t, EtagCase, typ, "Expected EtagCase type")
	assert.Equal(t, int64(12345), oid, "Expected OID to be 12345")
	assert.Equal(t, int32(0), ver, "Expected version to be 0")
	t.Logf("Successfully decoded ETag with Type: %v, OID: %d, Version: %d", typ, oid, ver)

	_, _, _, err = DecodeEtag("invalidEtag")
	assert.Error(t, err, "Decoding invalid ETag should return an error")
	t.Log("Decoding invalid ETag returned error as expected")
}

// TestExpectEtag tests the ExpectEtag function
func TestExpectEtag(t *testing.T) {
	encoded, _ := EncodeEtag(EtagCase, 12345, 0)
	tag, err := ExpectEtag(EtagCase, encoded)
	assert.NoError(t, err, "ExpectEtag should succeed for valid input")
	assert.Equal(t, int64(12345), tag.Oid, "Expected OID to be 12345")
	t.Log("ExpectEtag successfully returned the correct tag for valid input")

	_, err = ExpectEtag(EtagCase, "invalidEtag")
	assert.Error(t, err, "ExpectEtag should return error for invalid input")
	t.Log("ExpectEtag returned error for invalid input as expected")

	encodedWrongType, _ := EncodeEtag(EtagCaseLink, 12345, 0)
	_, err = ExpectEtag(EtagCase, encodedWrongType)
	assert.Error(t, err, "ExpectEtag should return error for mismatched types")
	t.Log("ExpectEtag returned error for mismatched types as expected")
}

// TestEtagOrId tests the EtagOrId function
func TestEtagOrId(t *testing.T) {
	// Test ETag input
	encoded, _ := EncodeEtag(EtagCase, 12345, 0)
	tag, err := EtagOrId(EtagCase, encoded)
	assert.NoError(t, err, "EtagOrId should succeed for valid ETag input")
	assert.Equal(t, int64(12345), tag.Oid, "Expected OID to be 12345")
	t.Log("EtagOrId returned valid tag for ETag input")

	// Test OID input
	tag, err = EtagOrId(EtagCase, "12345")
	assert.NoError(t, err, "EtagOrId should succeed for valid OID input")
	assert.Equal(t, int64(12345), tag.Oid, "Expected OID to be 12345")
	t.Log("EtagOrId returned valid tag for OID input")

	// Test invalid OID input
	_, err = EtagOrId(EtagCase, "invalidOid")
	assert.Error(t, err, "EtagOrId should return error for invalid OID input")
	t.Log("EtagOrId returned error for invalid OID input as expected")

	// Test mismatched types
	encodedWrongType, _ := EncodeEtag(EtagCaseLink, 12345, 0)
	_, err = EtagOrId(EtagCase, encodedWrongType)
	assert.Error(t, err, "EtagOrId should return error for mismatched types")
	t.Log("EtagOrId returned error for mismatched types as expected")
}

// TestGetTag tests GetTag function
func TestGetTag(t *testing.T) {
	node := mockVersional{ID: "12345", Ver: 0}
	tag, err := GetTag(node, EtagCase)
	assert.NoError(t, err, "GetTag should succeed for valid node")
	assert.Equal(t, int64(12345), tag.Oid, "Expected OID to be 12345")
	assert.Equal(t, int32(0), tag.GetVer(), "Expected version to be 0")
	t.Log("GetTag successfully returned the correct tag for valid node")
}

// mockVersional is a mock implementation of the IVersional interface.
type mockVersional struct {
	ID  string
	Ver int32
}

// ProtoReflect implements IVersional.
func (m mockVersional) ProtoReflect() protoreflect.Message {
	panic("unimplemented")
}

func (m mockVersional) GetId() string {
	return m.ID
}

func (m mockVersional) GetVer() int32 {
	return m.Ver
}
