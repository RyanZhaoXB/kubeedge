package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var config Configure
var once sync.Once

type Configure struct {
	v1alpha1.DeviceManager
}

func InitConfigure(deviceManager *v1alpha1.DeviceManager) {
	once.Do(func() {
		config = Configure{
			DeviceManager: *deviceManager,
		}
	})
}

func Get() *Configure {
	return &config
}
