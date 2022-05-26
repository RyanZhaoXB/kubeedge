package mapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager"
	"github.com/kubeedge/kubeedge/edge/pkg/devicemanager/utils/udsclient"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
)

// MapperService implements the mapper interface of DMI
type MapperService struct {
}

func newMapperService() *MapperService {
	return &MapperService{}
}

func (ms MapperService) MapperRegister(mapper *dmiapi.MapperInfo) error {
	devicemanager.MapperInfos[mapper.Name] = mapper
	klog.Infof("mapper list: %v", devicemanager.MapperInfos)
	return nil
}

func (ms MapperService) GetMapper(mapperName string) (*dmiapi.MapperInfo, error) {
	//TODO: get mapper through uds
	url := "http://test.sock/v1/kubeedge/mapper/" + mapperName
	data, err := httpRequest(url, http.MethodGet, nil)
	if err != nil {
		klog.Errorf("fail to get mapper info with error : %+v", err)
		return nil, err
	}

	var mapper dmiapi.MapperInfo
	if err = json.Unmarshal(data, &mapper); err != nil {
		klog.Errorf("fail to unmarshal mapper info with error : %+v", err)
		return nil, err
	}
	devicemanager.MapperInfos[mapperName] = &mapper
	return &mapper, nil
}

func (ms MapperService) HealthCheck(mapperName string) (string, error) {
	//TODO: healthcheck mapper through uds
	url := "http://test.sock/v1/kubeedge/mapper/" + mapperName + "/health"
	data, err := httpRequest(url, http.MethodGet, nil)
	if err != nil {
		klog.Errorf("fail to get mapper healthcheck with error : %+v", err)
		return "", err
	}

	return string(data), nil
}

func httpRequest(url, method string, body []byte) ([]byte, error){
	client := udsclient.NewHTTPClient(udsclient.SockPath)
	req, err := udsclient.BuildRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		klog.Errorf("fail to build request with error : %+v", err)
		return nil, err
	}
	res, err := udsclient.SendRequest(req, client)
	if err != nil {
		klog.Errorf("fail to send request with error : %+v", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		klog.Errorf("fail to get resp with code : %d", res.StatusCode)
		return nil, fmt.Errorf("fail to get mapper info")
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		klog.Errorf("fail to read data with error : %+v", err)
		return nil, err
	}

	return data, nil
}