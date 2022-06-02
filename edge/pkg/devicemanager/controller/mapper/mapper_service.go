package mapper

import (
	"encoding/json"
	"net/http"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/utils/udsclient"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
)

var MapperInfos map[string]*dmiapi.MapperInfo

// MapperService implements the mapper interface of DMI
type MapperService struct {
}

func init() {
	MapperInfos = make(map[string]*dmiapi.MapperInfo)
}

func newMapperService() *MapperService {
	return &MapperService{}
}

func (ms MapperService) MapperRegister(mapper *dmiapi.MapperInfo) error {
	MapperInfos[mapper.Name] = mapper
	klog.Infof("mapper list: %v", MapperInfos)
	return nil
}

func (ms MapperService) GetMapper(mapperName string) (*dmiapi.MapperInfo, error) {
	//TODO: get mapper through uds
	url := "http://test.sock/v1/kubeedge/mapper/" + mapperName
	data, err := udsclient.HttpRequest(url, http.MethodGet, nil)
	if err != nil {
		klog.Errorf("fail to get mapper info with error : %+v", err)
		return nil, err
	}

	var mapper dmiapi.MapperInfo
	if err = json.Unmarshal(data, &mapper); err != nil {
		klog.Errorf("fail to unmarshal mapper info with error : %+v", err)
		return nil, err
	}
	MapperInfos[mapperName] = &mapper
	return &mapper, nil
}

func (ms MapperService) HealthCheck(mapperName string) (string, error) {
	//TODO: healthcheck mapper through uds
	url := "http://test.sock/v1/kubeedge/mapper/" + mapperName + "/health"
	data, err := udsclient.HttpRequest(url, http.MethodGet, nil)
	if err != nil {
		klog.Errorf("fail to get mapper healthcheck with error : %+v", err)
		return "", err
	}

	return string(data), nil
}
