package dmi

import (
	"github.com/emicklei/go-restful"

	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

// DeviceManagerService interface should be implemented by a device mapper or edgecore.
// The methods should be thread-safe.
type DeviceManagerService interface {
	DeviceMapperManager
	DeviceManager
	DeviceUpgradeManager
	DeviceEventManager
	DeviceCommandManager
	DeviceDataManager
}

// DeviceMapperManager contains methods for mapper name, version and API version.
type DeviceMapperManager interface {
	// GetMapper returns the device mapper name, device mapper version and device mapper API version
	GetMapper(mapperName string) (*dmiapi.MapperInfo, error)

	HealthCheck(mapperName string) (string, error)

	MapperRegister(mapper *dmiapi.MapperInfo) error
}

// DeviceManager contains methods to manipulate devices managed by a
// device mapper. The methods are thread-safe.
type DeviceManager interface {

	// CreateDevice creates a new device.
	CreateDevice(config *dmiapi.DeviceConfig) (string, error)

	// RemoveDevice removes the device from platform by device name.
	RemoveDevice(deviceName string) error

	// UpdateDevice update device meta data.
	UpdateDevice(deviceName string, config *dmiapi.DeviceConfig) error

	// UpdateDeviceStatus updates the status of the device.
	UpdateDeviceStatus(deviceName string, desiredDevice *v1alpha2.DeviceStatus) error

	// ReportDeviceStatus updates the reported status of the device from mapper.
	ReportDeviceStatus(deviceName string, reportedDevice *v1alpha2.DeviceStatus) error

	// PatchDeviceStatus patches the status of the device. it will be implemented later.
	PatchDeviceStatus(deviceName string, desiredDevice *v1alpha2.DeviceStatus) error

	// ListDevices lists all devices by filters.
	ListDevices(filter *dmiapi.DeviceFilter) ([]*v1alpha2.DeviceStatus, error)

	// GetDevice returns the status of the device.
	GetDevice(deviceName string) (*v1alpha2.DeviceStatus, error)
}

// DeviceDataManager contains methods for accessing to the data of the device
type DeviceDataManager interface {
	// GetDeviceData returns the data of the specific properties from the reported value of twins
	GetDeviceData(deviceName string, propertyNames []string) ([]v1alpha2.Twin, error)

	// ListDeviceData lists all properties of a device from the reported value of twins
	ListDeviceData(deviceName string) ([]v1alpha2.Twin, error)

	// PublishDeviceData publish the data of properties to pubList urls from the reported value of twins
	PublishDeviceData(deviceName string, twins []v1alpha2.Twin, pubList []string) (v1alpha2.Twin, error)

	// UpdateDeviceData updates the data of properties from the desired value of twins
	UpdateDeviceData(deviceName string, twins []v1alpha2.Twin) (v1alpha2.Twin, error)
}

// DeviceUpgradeManager contains methods for upgrading devices
type DeviceUpgradeManager interface {
	// CheckUpgrade checks whether a device can be upgraded
	CheckUpgrade(deviceName string) (string, error)

	// UpgradeDevice upgrades a device
	UpgradeDevice(deviceName string, upgradeUrl string, version string) error
}

// DeviceCommandManager contains methods for accessing to the command of the device
type DeviceCommandManager interface {
	// ListCommands list all the commands of a device
	ListCommands(deviceName string) ([]dmiapi.CommandInfo, error)

	// GetCommand get the information a command of a device
	GetCommand(deviceName string, commandName string) (dmiapi.CommandInfo, error)

	// ExecCommand execute a command of a device
	ExecCommand(deviceName string, commandName string, body dmiapi.CommandBody) (*restful.Response, error)
}

// DeviceEventManager contains methods for accessing to the device event
type DeviceEventManager interface {

	// GetDeviceEvent get the information of an event of a device
	GetDeviceEvent(deviceName string, eventID string) (dmiapi.Event, error)

	// ListDeviceEvent get the list of events of a device
	ListDeviceEvent(deviceName string, filter dmiapi.EventFilter) ([]dmiapi.Event, error)

	// GetMapperEvent get the information of an event of a mapper
	GetMapperEvent(mapperName string) (dmiapi.Event, error)

	// ListMapperEvent get the list of events of a mapper
	ListMapperEvent(mapperName string, filter dmiapi.EventFilter) ([]dmiapi.Event, error)

	// CreateEvent creates an event to system
	CreateEvent(deviceName string, event dmiapi.Event) error
}
