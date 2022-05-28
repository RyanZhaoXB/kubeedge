package mappers

import (
	"github.com/emicklei/go-restful"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
)

func (d *DeviceTwinShim) ListCommands(deviceID string, deviceName string) ([]dmiapi.CommandInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) GetCommand(deviceID string, deviceName string, commandName string) (dmiapi.CommandInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) ExecCommand(deviceID string, deviceName string, commandName string, body dmiapi.CommandBody) (*restful.Response, error) {
	//TODO implement me
	panic("implement me")
}
