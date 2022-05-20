package udsserver

import (
	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"
)

// StartServer serves
func StartServer(address string, container *restful.Container) {
	uds := NewUnixDomainSocket(address)
	klog.Info("start unix domain socket server")
	if err := uds.StartServer(container); err != nil {
		klog.Exitf("failed to start uds server: %v", err)
		return
	}
}
