package etag

import (
	"fmt"
	"strconv"
)

// GetOid parses the given id string into a serial unique OID identifier.
func GetOid(typ EtagType, id string) (oid int64, err error) {
	defer func() {
		if err != nil {
			oid = 0 // NULLify
		}
	}()
	oid, err = strconv.ParseInt(id, 10, 64)
	if err != nil || oid < 1 {
		err = fmt.Errorf("node( id:%s ); expected: positive int64", id)
		return
	}
	return
}

// GetId formats the given serial OID identifier into a string presentation.
func GetId(typ EtagType, oid int64) (id string, err error) {
	if oid < 1 {
		err = fmt.Errorf("node( id:%d ); expected: positive int64", oid)
		return
	}
	return strconv.FormatInt(oid, 10), nil
}

// MustId ensures that the ID is valid or panics if it is not.
func MustId(typ EtagType, id string, err error) string {
	if err == nil && id == "" {
		err = fmt.Errorf("node( id: ); required")
	}
	if err != nil {
		panic(err)
	}
	return id
}
