package mappers

import dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"

func (d *DeviceTwinShim) GetDeviceEvent(deviceID string, deviceName string, eventID string) (dmiapi.Event, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) ListDeviceEvent(deviceID string, deviceName string, filter dmiapi.EventFilter) ([]dmiapi.Event, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) GetMapperEvent(mapperName string) (dmiapi.Event, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) ListMapperEvent(mapperName string, filter dmiapi.EventFilter) ([]dmiapi.Event, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) CreateEvent(deviceID string, deviceName string, event dmiapi.Event) error {
	//TODO implement me
	panic("implement me")
}
