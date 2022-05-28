package mappers

import (
	"context"
	"errors"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
	"github.com/kubeedge/kubeedge/mappers/config"
	"github.com/kubeedge/kubeedge/mappers/dtclient"
	"github.com/kubeedge/kubeedge/mappers/dttype"
)

type DeviceTwinShim struct {
	GroupID     string
	NodeName    string
	cfg         *config.MapperConfig
	ctx         *context.Context
	deviceList  *sync.Map
	deviceMutex *sync.Map
	mutex       *sync.RWMutex
	// TODO remove
	CommChan map[string]chan interface{}
}

func (d *DeviceTwinShim) Name() string {
	return "device_twin"
}

// getMutex get mutex
func (d *DeviceTwinShim) getMutex(deviceID string) (*sync.Mutex, bool) {
	v, mutexExist := d.deviceMutex.Load(deviceID)
	if !mutexExist {
		klog.Errorf("getMutex device %s not exist", deviceID)
		return nil, false
	}
	mutex, isMutex := v.(*sync.Mutex)
	if isMutex {
		return mutex, true
	}
	return nil, false
}

// lock the device
func (d *DeviceTwinShim) lock(deviceID string) bool {
	deviceMutex, ok := d.getMutex(deviceID)
	if ok {
		d.mutex.RLock()
		deviceMutex.Lock()
		return true
	}
	return false
}

// unlock remove the lock of the device
func (d *DeviceTwinShim) unlock(deviceID string) bool {
	deviceMutex, ok := d.getMutex(deviceID)
	if ok {
		deviceMutex.Unlock()
		d.mutex.RUnlock()
		return true
	}
	return false
}

// lockAll get all lock
func (d *DeviceTwinShim) lockAll() {
	d.mutex.Lock()
}

// unlockAll release all lock
func (d *DeviceTwinShim) unlockAll() {
	d.mutex.Unlock()
}

// isDeviceExist judge device is exists
func (d *DeviceTwinShim) isDeviceExist(deviceID string) bool {
	_, ok := d.deviceList.Load(deviceID)
	return ok
}

// getDeviceInstance get device
func (d *DeviceTwinShim) getDeviceInstance(deviceID string) (*dttype.Device, bool) {
	device, ok := d.deviceList.Load(deviceID)
	if ok {
		if device, isDevice := device.(*dttype.Device); isDevice {
			return device, true
		}
		return nil, false
	}
	return nil, false
}

// syncDeviceFromSqlite sync device from sqlite
func (d *DeviceTwinShim) syncDeviceFromSqlite(deviceID string) error {
	klog.V(2).Infof("Sync device detail info from DB of device %s", deviceID)
	if _, exist := d.getDeviceInstance(deviceID); !exist {
		d.deviceMutex.Store(deviceID, &sync.Mutex{})
	}

	devices, err := dtclient.QueryDevice("id", deviceID)
	if err != nil {
		klog.Errorf("query device failed, id: %s, err: %v", deviceID, err)
		return err
	}
	if len(*devices) == 0 {
		return errors.New("not found device")
	}
	device := (*devices)[0]

	deviceAttr, err := dtclient.QueryDeviceAttr("deviceid", deviceID)
	if err != nil {
		klog.Errorf("query device attr failed, id: %s, err: %v", deviceID, err)
		return err
	}

	deviceTwin, err := dtclient.QueryDeviceTwin("deviceid", deviceID)
	if err != nil {
		klog.Errorf("query device twin failed, id: %s, err: %v", deviceID, err)
		return err
	}

	d.deviceList.Store(deviceID, &dttype.Device{
		ID:          deviceID,
		Name:        device.Name,
		Description: device.Description,
		State:       device.State,
		LastOnline:  device.LastOnline,
		Attributes:  dttype.DeviceAttrToMsgAttr(*deviceAttr),
		Twin:        dttype.DeviceTwinToMsgTwin(*deviceTwin),
	})

	return nil
}

// commTo communicate
func (d *DeviceTwinShim) commTo(dtmName string, content interface{}) error {
	if v, exist := d.CommChan[dtmName]; exist {
		v <- content
		return nil
	}
	return errors.New("not found chan to communicate")
}

// send result
func (d *DeviceTwinShim) send(identity string, action string, module string, msg *model.Message) error {
	dtMsg := &dttype.DTMessage{
		Action:   action,
		Identity: identity,
		Type:     module,
		Msg:      msg}
	return d.commTo(module, dtMsg)
}

// buildModelMessage build mode message
func (d *DeviceTwinShim) buildModelMessage(group string, parentID string, resource string, operation string, content interface{}) *model.Message {
	msg := model.NewMessage(parentID)
	msg.BuildRouter(modules.TwinGroup, group, resource, operation)
	msg.Content = content
	return msg
}

func (d *DeviceTwinShim) GetMapper(mapperName string) (*dmiapi.MapperInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) HealthCheck(mapperName string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) MapperRegister(mapper *dmiapi.MapperInfo) error {
	//TODO implement me
	panic("implement me")
}
