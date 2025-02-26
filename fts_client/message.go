package fts_client

import (
	"encoding/json"
	"fmt"
)

const (
	MessageCreate = "create"
	MessageUpdate = "update"
	MessageDelete = "delete"
)

type MessageId string

type Message struct {
	Id         MessageId       `json:"id"`
	DomainId   int64           `json:"domain_id,omitempty"`
	ObjectName string          `json:"object_name,omitempty"`
	Date       int64           `json:"date,omitempty"`
	Body       json.RawMessage `json:"body,omitempty"`
	data       []byte
}

func (id *MessageId) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("cannot unmarshal MessageId: %w", err)
	}

	switch raw.(type) {
	case float64:
		raw = int64(raw.(float64))
	}

	*id = MessageId(fmt.Sprintf("%v", raw))
	return nil
}

func NewMessageId(v any) MessageId {
	return MessageId(fmt.Sprintf("%v", v))
}

func NewMessageJSON(domainId int64, objectName string, id any, row any) ([]byte, error) {
	var err error
	m := Message{
		Id:         NewMessageId(id),
		DomainId:   domainId,
		ObjectName: objectName,
		Body:       nil,
	}
	if row != nil {
		m.Body, err = json.Marshal(row)
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(m)
}
