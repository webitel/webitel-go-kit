package model

import (
	"encoding/json"
	"time"
)

type Record struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Service   Service           `json:"service"`
	TraceId   string            `json:"trace_id,omitempty"`
	SpanId    string            `json:"span_id,omitempty"`
	Message   string            `json:"message"`
	Context   map[string]string `json:"context,omitempty"`
	Error     *ErrorType        `json:"error,omitempty"`
}

type Service struct {
	Id      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Build   int    `json:"build,omitempty"`
}

type ErrorType struct {
	Id         string `json:"id"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
	StackTrace string `json:"stack_trace,omitempty"`
}

func (r *Record) Jsonify() ([]byte, error) {
	return json.Marshal(r)
}
