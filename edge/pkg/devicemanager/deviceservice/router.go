package deviceservice

import (
	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"
)

func (ds *DeviceService) registerRoutes(ws *restful.WebService) {
	ws.Route(ws.GET("/device").To(ds.connect))
}

func (ds *DeviceService) connect(r *restful.Request, w *restful.Response) {
	klog.Infof("uds server receives context: %s", r)
	w.Write([]byte("hello client aaaa"))
	return
}
