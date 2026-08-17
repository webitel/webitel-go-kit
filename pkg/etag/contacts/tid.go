package contacts

import (
	"fmt"
	"strconv"
)

// The [T]uple [ID]entifier is a pointer
// to a specific version of the unique tuple.
type Tid struct {
	// REQUIRED. Tuple unique identifier.
	// Positive non-zero integer number.
	Oid int64
	// OPTIONAL. Tuple revision number.
	// Zero-based integer sequence number.
	// REQUIRED. As a part of ETag identifier.
	Ver *int32
}

// GetId returns string format of e.Oid identifier.
func (e *Tid) GetId() string {
	id, _ := GetId(e.GetOid())
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

// IsVer reports whether e.Ver is associated and equals num.
func (e *Tid) IsVer(num int32) bool {
	if e.HasVer() {
		return *(e.Ver) == num
	}
	return false
}

// IsNone reports whether e points to no tuple.
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

// GetOid parses the given id string into a serial unique OID identifier.
func GetOid(id string) (int64, error) {
	oid, err := strconv.ParseInt(id, 10, 64)
	if err != nil || oid < 1 {
		return 0, fmt.Errorf("node( id:%s ); expected: positive int64", id)
	}
	return oid, nil
}

// GetId formats the given serial OID identifier into a string presentation.
func GetId(oid int64) (string, error) {
	if oid < 1 {
		return "", fmt.Errorf("node( id:%d ); expected: positive int64", oid)
	}
	return strconv.FormatInt(oid, 10), nil
}
