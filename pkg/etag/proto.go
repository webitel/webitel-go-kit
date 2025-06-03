package etag

import (
	"errors"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoMessage reflects:
// - google.golang.org/protobuf/proto.Message
// - github.com/golang/protobuf/proto.MessageV2() protoV2.Message
type ProtoMessage = protoreflect.ProtoMessage

// INode interface represents single data node within contacts package schema.
type INode interface {
	// GetId MUST return an unique identifier of this data node.
	GetId() string

	// FIXME: Accept: proto.Message(s)
	ProtoMessage
}

// IVersional interface
type IVersional interface {
	// MUST implements INode
	INode
	// GetVertion returns latest revision(version) number of this data node.
	GetVer() int32
}

// IRemovable interface
type IRemovable interface {
	// Positive timestamp indicates that this source
	// was temporary removed from main view.
	GetDeletedAt() int64
}

type IList[TNode INode] interface {
	GetData() []TNode
}

type IVisitor[TNode INode] interface {
	// Check given node for data integrity violations.
	Constraint(input TNode) error
	// Check given nodes for violations of data uniqueness.
	UniqueConstraint(input, exist TNode) error
}

// Constraint checks given this INode for data integrity violations.
type Constraint[TNode INode] func(this TNode) error

// RequireId constraint
// RequireId ensures that the node's ID is not empty, returns an error if missing.
func RequireId[TNode INode](nodeOf string, typ EtagType) Constraint[TNode] {
	if nodeOf == "" {
		nodeOf = "node"
	}
	return func(node TNode) error {
		if node.GetId() == "" {
			return errors.New("node id required")
		}
		return nil
	}
}

// UniqueConstraint checks given INode(s) for violations of data uniqueness.
type UniqueConstraint[TNode INode] func(this, that TNode) error

func UniqueId(this, that INode) error {
	if id := this.GetId(); id != "" && id == that.GetId() {
	}
	return nil
}
