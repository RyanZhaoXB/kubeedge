package deviceservice

import (
	"github.com/emicklei/go-restful"
	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/controller/mapper"
	"k8s.io/klog/v2"
)

func (ds *DeviceService) registerRoutes(ws *restful.WebService) {
	ws.Route(ws.GET("/device").To(ds.connect))
	ws.Route(ws.POST("/mapper/register").To(mapper.NewMapperController.RegisterMapper))
	ws.Route(ws.GET("/mapper/{mapper_name}").To(mapper.NewMapperController.GetMapper))
	ws.Route(ws.GET("/mapper/{mapper_name}/healthcheck").To(mapper.NewMapperController.MapperHealthCheck))
}

func (ds *DeviceService) connect(r *restful.Request, w *restful.Response) {
	klog.Infof("uds server receives context: %s", r)
	w.Write([]byte("hello"))
	return
}
