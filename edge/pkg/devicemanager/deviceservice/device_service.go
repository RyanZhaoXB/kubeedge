package deviceservice

import "github.com/emicklei/go-restful"

type DeviceService struct {
	Container *restful.Container
}

func NewDeviceService() *DeviceService {
	return &DeviceService{
		Container: restful.NewContainer(),
	}
}

func (ds *DeviceService) InstallDefaultHandler(rootPath string) {
	ws := new(restful.WebService)
	ws.Path(rootPath).Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	ds.registerRoutes(ws)
	ds.Container.Add(ws)
}
