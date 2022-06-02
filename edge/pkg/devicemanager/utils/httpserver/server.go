package httpserver

import (
	"github.com/emicklei/go-restful"
	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/config"
	"k8s.io/klog/v2"
	"net"
	"net/http"
)

// StartServer serves
func StartServer(container *restful.Container) {
	klog.Info("start device manager server")
	server := http.Server{
		Addr: net.JoinHostPort(config.Address, config.Port),
		Handler: container,
	}
	if err := server.ListenAndServeTLS("", ""); err != nil {
		klog.Exitf("failed to start device manager server: %v", err)
		return
	}
}
