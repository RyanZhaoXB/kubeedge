package constants

// Constants for protocol, datatype, configmap, deviceProfile
const (
	OPCUA              = "opcua"
	Modbus             = "modbus"
	Bluetooth          = "bluetooth"
	CustomizedProtocol = "customized-protocol"

	DataTypeInt     = "int"
	DataTypeInteger = "integer"
	DataTypeString  = "string"
	DataTypeDouble  = "double"
	DataTypeFloat   = "float"
	DataTypeBoolean = "boolean"
	DataTypeBytes   = "bytes"

	DeviceProfileConfigPrefix = "device-profile-config-"

	DeviceProfileJSON = "deviceProfile.json"
)

// ResourceTypeDeviceModel is type for device model distribute to node
const (
	ResourceTypeDeviceModel  = "devicemodel"
	ResourceTypeDevice       = "device"
	ResourceTypeDeviceMapper = "devicemapper"
)

const UnixNetworkType = "unix"
