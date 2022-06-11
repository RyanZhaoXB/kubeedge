package mappers

import "github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"

func (d *DeviceTwinShim) GetDeviceData(deviceName string, propertyNames []string) ([]v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) ListDeviceData(deviceName string) ([]v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) PublishDeviceData(deviceName string, twins []v1alpha2.Twin, pubList []string) (v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) UpdateDeviceData(deviceName string, twins []v1alpha2.Twin) (v1alpha2.Twin, error) {
	//TODO implement me
	panic("implement me")
}
