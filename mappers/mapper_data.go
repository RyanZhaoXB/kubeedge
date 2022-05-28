package mappers

import "github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"

func (d *DeviceTwinShim) GetDeviceData(deviceID string, deviceName string, propertyNames []string) ([]v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) ListDeviceData(deviceID string, deviceName string) ([]v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) PublishDeviceData(deviceID string, deviceName string, twins []v1alpha2.Twin, pubList []string) (v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) UpdateDeviceData(deviceID string, deviceName string, twins []v1alpha2.Twin) (v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}
