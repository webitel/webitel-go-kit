package contacts

import (
	"encoding/base32"
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/webitel/webitel-go-kit/pkg/etag"
)

// Base32 alphabet for internal ETag string presentation.
// NOTE: excludes digits so a raw numeric OID can never parse as an ETag.
// Frozen: must stay identical to the contacts service encoding.
const encodeEtag = "eNYuSrBqFdCJILGfPtobMQaWEyTzlXmD"

// ETagEncoding is base32.Encoding for human-readable text presentation of internal ETag values.
var ETagEncoding = base32.NewEncoding(encodeEtag).WithPadding(base32.NoPadding)

// AppendTag appends the (ver, typ, oid) tuple to dst as protobuf varints.
func AppendTag(dst []byte, typ Type, oid int64, ver int32) []byte {
	dst = protowire.AppendVarint(dst, uint64(ver))
	dst = protowire.AppendVarint(dst, uint64(typ))
	dst = protowire.AppendVarint(dst, uint64(oid))
	return dst
}

// EncodeEtag encodes the (typ, oid, ver) tuple into an opaque ETag string.
func EncodeEtag(typ Type, oid int64, ver int32) (string, error) {
	if typ <= NoType {
		return "", fmt.Errorf("etag: encode tag{typ:%d}; expect: positive, non-zero integer identifier", int8(typ))
	}
	if oid < 1 {
		return "", fmt.Errorf("etag: encode tag{oid:%d}; expect: positive, non-zero integer identifier", oid)
	}
	if ver < 0 {
		return "", fmt.Errorf("etag: encode tag{ver:%d}; expect: zero-based, positive integer number", ver)
	}
	buf := AppendTag(make([]byte, 0, 8), typ, oid, ver)
	return ETagEncoding.EncodeToString(buf), nil
}

const errTagMalformed = -1

// ConsumeTag parses the (ver, typ, oid) varint tuple from src.
// Negative n reports malformed input.
func ConsumeTag(src []byte) (typ Type, oid int64, ver int32, n int) {
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
	typ = Type(v[1])
	oid = int64(v[2])
	return
}

// DecodeEtag decodes an opaque ETag string back into the (typ, oid, ver) tuple.
func DecodeEtag(s string) (typ Type, oid int64, ver int32, err error) {
	src, err := ETagEncoding.DecodeString(s)
	if err != nil {
		err = fmt.Errorf("( etag:%s ); invalid encoding", s)
		return
	}
	var n int
	typ, oid, ver, n = ConsumeTag(src)
	if n <= errTagMalformed || n < len(src) {
		err = fmt.Errorf("( etag:%s ); malformed input", s)
		return
	}
	return
}

// DecodeTag decodes an ETag string into its type and Tid.
func DecodeTag(s string) (typ Type, tag Tid, err error) {
	var rev int32
	typ, tag.Oid, rev, err = DecodeEtag(s)
	if err != nil {
		return
	}
	tag.Ver = &rev
	return
}

// EncodeTag encodes the given Tid of the given type into an ETag string.
func EncodeTag(typ Type, tag Tid) (string, error) {
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

// EtagOrId parses the input as either a valid ETag of the expected type
// or a raw unique object identifier.
func EtagOrId(of Type, s string) (tag Tid, err error) {
	if !validType(of) {
		return Tid{}, fmt.Errorf("etag( typ:%d ); accept: positive, non-zero integer identifier", int8(of))
	}
	typ, tag, err := DecodeTag(s)
	if err == nil {
		if of != typ {
			return Tid{}, fmt.Errorf("( etag:%s ); illegal", s)
		}
		return tag, nil
	}
	tag = Tid{}
	tag.Oid, err = GetOid(s)
	if err != nil {
		return Tid{}, fmt.Errorf("( etag:%s ); invalid", s)
	}
	return tag, nil
}

// InputIdOrEtag parses the input set of id-or-etag strings.
// Duplicate identifiers are accepted, as the contacts service does.
func InputIdOrEtag(typeOf Type, input ...string) (data Tids, err error) {
	split := func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}
	input = etag.FieldsFunc(
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
	}
	return data, nil
}

// GetTag builds a Tid from the given versional node.
func GetTag(node etag.IVersional) (tag Tid, err error) {
	tag.Oid, err = GetOid(node.GetId())
	if err != nil {
		return
	}
	rev := node.GetVer()
	tag.Ver = &rev
	return
}

// MustTag ensures the tag is a complete ETag tuple or panics.
func MustTag(tag Tid, err error) Tid {
	if err == nil && !tag.HasOid() {
		err = fmt.Errorf("tag( oid: int64! ) ISNULL")
	}
	if err == nil && !tag.HasVer() {
		err = fmt.Errorf("tag( rev: int32! ) ISNULL")
	}
	if err != nil {
		panic(err)
	}
	return tag
}
