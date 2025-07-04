package etag

import (
	"encoding/base32"
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/encoding/protowire"
)

type Tag struct {
	// Encoded tuple identifier
	Tid
	// Encoded string identifier
	Id string
}

func (e *Tag) IsZero() bool {
	return e == nil || e.Oid == 0
}

func (e *Tag) IsEtag() bool {
	return !e.IsZero() && e.Ver != nil
}

// The [T]uple [ID]entifier is a pointer
// to a specific version of the unique tuple.
type Tid struct {
	// OPTIONAL. Tuple revision number.
	// Zero-based integer sequence number.
	// REQUIRED. As a part of ETag identifier.
	Ver *int32
	// REQUIRED. Tuple unique identifier.
	// Positive non-zero integer number.
	Oid int64
}

// GetId returns string format of the Oid identifier dynamically using the passed type.
func (e *Tid) GetId(typ EtagType) string {
	id, _ := GetId(typ, e.GetOid()) // Using dynamic type
	return id
}

// GetOid returns the tuple's unique integer identifier or zero.
func (e *Tid) GetOid() int64 {
	if e != nil && e.Oid > 0 {
		return e.Oid
	}
	return 0
}

// HasOid reports whether e is valid.
func (e *Tid) HasOid() bool {
	return e.GetOid() > 0
}

// HasVer reports whether the tuple version is associated.
// True means e was build from -or- can be used for ETag identifier.
func (e *Tid) HasVer() bool {
	return e != nil && e.Ver != nil
}

// GetVer returns the associated tuple version or zero.
func (e *Tid) GetVer() int32 {
	if e.HasVer() {
		return *(e.Ver)
	}
	return 0
}

// VerOf reports whether e.Ver associated and v equals to e.Ver.
func (e *Tid) IsVer(num int32) bool {
	if e.HasVer() {
		return *(e.Ver) == num
	}
	return false
}

// IsZero reports whether e points to any tuple.
func (e *Tid) IsNone() bool {
	return !e.HasOid()
}

func (e *Tid) Valid() error {
	if e == nil {
		return fmt.Errorf("missing tid")
	}
	if e.Oid < 1 {
		return fmt.Errorf("missing oid")
	}
	return nil
}

type Tids []Tid

func (e Tids) IsNone() bool {
	return len(e) > 0
}

func (e Tids) Oids() []int64 {
	if n := len(e); n > 0 {
		oids := make([]int64, n)
		for i, v := range e {
			oids[i] = v.GetOid()
		}
		return oids
	}
	return nil
}

// InputIdOrEtag parses <input> set of UNIQUE [id!].
func InputIdOrEtag(typeOf EtagType, input ...string) (data Tids, err error) {
	split := func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}
	input = FieldsFunc(
		input, func(input string) []string {
			return strings.FieldsFunc(input, split)
		},
	)
	if len(input) == 0 || (len(input) == 1 && input[0] == "") {
		return nil, nil
	}
	data = make(Tids, len(input))
	for r, s := range input {
		data[r], err = EtagOrId(typeOf, s)
		if err != nil {
			return nil, err
		}
		for e := 0; e < r; e++ {
			if data[r].Oid == data[e].Oid {
				return nil, fmt.Errorf("input( etag: %s, id: %d ); duplicate", s, data[r].Oid)
			}
		}
	}
	return data, nil
}

// ETag Object reference
type ETag struct {
	// Object id of the object
	Tid
	// Tuple id of the object
	EtagType
}

func (e *ETag) Valid() error {
	// Check if the ETag object itself is nil
	if e == nil {
		return fmt.Errorf("missing tag")
	}
	// Check if the Type is valid
	if e.EtagType <= NoType {
		return fmt.Errorf("missing type")
	}
	// Validate the Tid field
	return e.Tid.Valid()
}

func (e *ETag) String() (string, error) {
	return EncodeEtag(e.EtagType, e.Oid, e.GetVer())
}

func AppendTag(dst []byte, typ EtagType, oid int64, ver int32) []byte {
	dst = protowire.AppendVarint(dst, uint64(ver))
	dst = protowire.AppendVarint(dst, uint64(typ))
	dst = protowire.AppendVarint(dst, uint64(oid))
	return dst
}

const (
	errTagMalformed = -iota
	errTagNoType
	errTagNoOid
)

func ConsumeTag(src []byte) (typ EtagType, oid int64, ver int32, n int) {
	var (
		r int       // read
		v [3]uint64 // values
	)
	for e := 0; e < 3; e++ {
		v[e], r = protowire.ConsumeVarint(src[n:])
		if r < 0 {
			n = errTagMalformed
			return
		}
		n += r
	}
	ver = int32(v[0])
	typ = EtagType(v[1])
	oid = int64(v[2])
	return
}

// Base32 alphabet for internal ETag string presentation without numbers
const encodeEtag = "QRSTVWXYZabcdefghjklmnpqrstvwxyz"

// ETagEncoding is base32.Encoding for human-readable text presentation of internal ETag values
var ETagEncoding = base32.NewEncoding(encodeEtag).WithPadding(base32.NoPadding)

func EncodeEtag(typ EtagType, oid int64, ver int32) (string, error) {
	if typ <= NoType {
		return "", fmt.Errorf("etag: encode tag{typ:%d}; expect: positive, non-zero integer identifier", int8(typ))
	}
	if oid < 1 {
		return "", fmt.Errorf("etag: encode tag{oid:%d}; expect: positive, non-zero integer identifier", oid)
	}
	if ver < 0 {
		return "", fmt.Errorf("etag: encode tag{ver:%d}; expect: zero-based, positive integer number", ver)
	}
	buf := AppendTag(nil, typ, oid, ver)
	return ETagEncoding.EncodeToString(buf), nil
}

func DecodeEtag(s string) (typ EtagType, oid int64, ver int32, err error) {
	src, err := ETagEncoding.DecodeString(s)
	if err != nil {
		err = fmt.Errorf("( etag:%s ); invalid encoding", s)
		return
	}

	var n int
	typ, oid, ver, n = ConsumeTag(src) // Adjust to return all three values
	if n <= errTagMalformed || n < len(src) {
		err = fmt.Errorf("( etag:%s ); malformed input", s)
		return
	}
	return
}

func DecodeTag(s string) (typ EtagType, tag Tid, err error) {
	var rev int32
	typ, tag.Oid, rev, err = DecodeEtag(s) // Adjust to return three values from DecodeEtag
	if err != nil {
		return
	}
	tag.Ver = &rev
	return
}

func EncodeTag(typ EtagType, tag Tid) (string, error) {
	if !validType(typ) {
		return "", fmt.Errorf("etag( typ:%d ); accept: positive, non-zero integer", int8(typ))
	}
	if tag.IsNone() {
		return "", fmt.Errorf("etag( oid:%d ); expect: positive, non-zero integer", tag.Oid)
	}
	if !tag.HasVer() || tag.GetVer() < 0 {
		return "", fmt.Errorf("etag( ver: ); expect: zero-based, positive integer")
	}
	return EncodeEtag(typ, tag.Oid, tag.GetVer())
}

// ExpectETag parses a given string as an ETag string of the expected reference type.
func ExpectEtag(of EtagType, s string) (tag Tid, err error) {
	if !validType(of) {
		return Tid{}, fmt.Errorf("etag: expect tag{typ:%d}; must be positive, non-zero integer identifier", int8(of))
	}
	typ, tag, err := DecodeTag(s) // DecodeTag now returns 3 values
	if err == nil {
		if of != typ { // Compare the provided type with the decoded type
			err = fmt.Errorf("invalid ETag identifier for type %d", of)
		}
	}
	return
}

// EtagOrId parses the input as either a valid ETag or a unique object identifier.
func EtagOrId(of EtagType, s string) (tag Tid, err error) {
	typ, tag, err := DecodeTag(s)
	if err == nil {
		if of != typ {
			return tag, fmt.Errorf("( etag:%s ); illegal type", s)
		}
		return tag, nil
	}

	tag.Oid, err = GetOid(of, s)
	if err != nil {
		return tag, fmt.Errorf("( etag:%s ); invalid format", s)
	}
	if tag.Oid < 1 {
		return tag, fmt.Errorf("( id:%s ); negative value", s)
	}
	return tag, nil
}

// GetTag from given node dynamically uses the type
func GetTag(node IVersional, typ EtagType) (tag Tid, err error) {
	tag.Oid, err = GetOid(typ, node.GetId())
	if err != nil {
		return
	}
	rev := node.GetVer()
	tag.Ver = &rev
	return
}
