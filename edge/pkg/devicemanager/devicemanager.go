package devicemanager

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/config"
	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/deviceservice"
	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/utils/httpserver"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"

	_ "github.com/kubeedge/kubeedge/edge/pkg/devicemanager/controller/mapper"
)



type DeviceManager struct {
	enable   bool
}

var _ core.Module = (*DeviceManager)(nil)

func newDeviceManager(enable bool) *DeviceManager {
	return &DeviceManager{
		enable:   enable,
	}
}

// Register register DeviceManager
func Register(s *v1alpha1.DeviceManager) {
	config.InitConfigure(s)
	core.Register(newDeviceManager(s.Enable))
}

func (dm *DeviceManager) Name() string {
	return modules.DeviceManagerModuleName
}

func (dm *DeviceManager) Group() string {
	return modules.DeviceGroup
}

func (dm *DeviceManager) Enable() bool {
	return dm.enable
}

func (dm *DeviceManager) Start() {
	dm.runDeviceManager()
}

func (dm *DeviceManager) runDeviceManager() {
	//SockPath := "unix://root/data/test.sock"
	rootPath := "/v1/kubeedge"
	deviceServer := deviceservice.NewDeviceService()
	deviceServer.InstallDefaultHandler(rootPath)
	go httpserver.StartServer(deviceServer.Container)
}
