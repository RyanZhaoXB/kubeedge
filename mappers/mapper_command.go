package mappers

import (
	"github.com/emicklei/go-restful"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
)

func (d *DeviceTwinShim) ListCommands(deviceName string) ([]dmiapi.CommandInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) GetCommand(deviceName string, commandName string) (dmiapi.CommandInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DeviceTwinShim) ExecCommand(deviceName string, commandName string, body dmiapi.CommandBody) (*restful.Response, error) {
	//TODO implement me
	panic("implement me")
}
