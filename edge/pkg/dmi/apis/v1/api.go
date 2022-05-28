package v1

import (
	"time"

	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

type MapperAccessMode string

// Access mode constants for a device property.
const (
	MQTT MapperAccessMode = "MQTT"
	REST MapperAccessMode = "REST"
)

type DeviceConfig struct {
	Model  v1alpha2.DeviceModel `json:"model"`
	Device v1alpha2.Device      `json:"device"`
}

type DeviceFilter struct {
	LabelSelector map[string]string
	DeviceID      string
	DeviceName    string
	State         string
	MapperID      string
	MapperName    string
}

type MapperInfo struct {
	// Name of the device mapper.
	Name string `json:"name,omitempty"`
	// Version of the device mapper. The string must be
	// semver-compatible.
	Version string `json:"version,omitempty"`
	// API version of the device mapper. The string must be
	// semver-compatible.
	ApiVersion string `json:"api_version,omitempty"`

	Protocol string	`json:"protocol,omitempty"`

	AccessMode MapperAccessMode	`json:"access_mode,omitempty"`
	// for mqtt or rest
	Address interface{}	`json:"address,omitempty"`

	State string	`json:"state,omitempty"`
}

type Event struct {
	BaseMessage
	message string
}

// BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

type EventFilter struct {
	deviceID   string
	deviceName string
	mapperName string
	timeBefore time.Time
	timeAfter  time.Time
}

type CommandInfo struct {
	Name       string
	DeviceName string
	Method     string
	Url        string
	Version    string
	Parameters map[string]string
}

type CommandBody struct {
}
