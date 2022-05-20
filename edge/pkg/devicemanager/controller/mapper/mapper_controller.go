package mapper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager"
	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/utils"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
)

type MapperController struct {
	mapperService *MapperService
}

// NewMapperController new mapper controller
var NewMapperController = MapperController{
	mapperService: newMapperService(),
}

func (mc MapperController) RegisterMapper(r *restful.Request, w *restful.Response) {
	reqBody, err := ioutil.ReadAll(r.Request.Body)
	if err != nil {
		klog.Errorf("Failed to read response body with err: %v", err)
		utils.WriteHttpResp(w, http.StatusBadRequest, nil)
		return
	}
	var mapper dmiapi.MapperInfo
	if err = json.Unmarshal(reqBody, &mapper); err != nil {
		klog.Errorf("Failed to Unmarshal reqBody with err: %v ", err)
		utils.WriteHttpResp(w, http.StatusBadRequest, nil)
		return
	}
	err = mc.mapperService.MapperRegister(&mapper)
	if err != nil {
		utils.WriteHttpResp(w, http.StatusBadRequest, nil)
		klog.Errorf("Failed to register mapper with err: %v", err)
		return
	}
	klog.Infof("mapper: %+v", devicemanager.MapperInfos)
	utils.WriteHttpResp(w, http.StatusOK, "success to register mapper")
	return
}

func (mc MapperController) GetMapper(r *restful.Request, w *restful.Response) {
	mapperName := r.PathParameter("mapper_name")
	mapperInfo, err := mc.mapperService.GetMapper(mapperName)
	if err != nil {
		utils.WriteHttpResp(w, http.StatusBadRequest, nil)
		klog.Errorf("Failed to get mapper with err: %v", err)
		return
	}
	utils.WriteHttpResp(w, http.StatusOK, mapperInfo)
	return
}

func (mc MapperController) MapperHealthCheck(r *restful.Request, w *restful.Response) {
	mapperName := r.PathParameter("mapper_name")
	mapperHealth, err := mc.mapperService.HealthCheck(mapperName)
	if err != nil {
		utils.WriteHttpResp(w, http.StatusBadRequest, nil)
		klog.Errorf("Failed to healthcheck mapper with err: %v", err)
		return
	}
	utils.WriteHttpResp(w, http.StatusOK, mapperHealth)
	return
}
