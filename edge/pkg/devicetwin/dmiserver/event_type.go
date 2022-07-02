package dmiserver

import (
	"time"
)

const (
	PropertyType string = "propertyType"
)

// MsgTwin the structure of device twin.
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

// TwinValue the structure of twin value.
type TwinValue struct {
	Value    *string       `json:"value,omitempty"`
	Metadata ValueMetadata `json:"metadata,omitempty"`
}

// ValueMetadata the meta of value.
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

// TypeMetadata the meta of value type.
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

// TwinVersion twin version.
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

// DeviceTwinUpdate the structure of device twin update.
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

// BaseMessage the base structure of event message.
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

// getTimestamp get current timestamp.
func getTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}
