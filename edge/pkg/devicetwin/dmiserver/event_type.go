package dmiserver

import (
	"time"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
)

const (
	PropertyType = "propertyType"
)

// DeviceTwinUpdate the structure of device twin update.
type DeviceTwinUpdate struct {
	types.BaseMessage
	Twin map[string]*types.MsgTwin `json:"twin"`
}

// getTimestamp get current timestamp.
func getTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}
