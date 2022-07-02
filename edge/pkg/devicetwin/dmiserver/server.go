package dmiserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	pb "github.com/kubeedge/kubeedge/edge/pkg/apis/dmi/v1"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

const (
	SockPath                 string = "/tmp/dmi.sock"
	UnixNetworkType          string = "unix"
	ResourceTypeDeviceMapper        = "devicemapper"
)

type server struct {
	mapperList      map[string]*pb.MapperInfo
	deviceList      map[string]*v1alpha2.Device
	deviceModelList map[string]*v1alpha2.DeviceModel
}

func (s *server) MapperRegister(ctx context.Context, in *pb.MapperRegisterRequest) (*pb.MapperRegisterResponse, error) {
	klog.Infof("receive mapper register: %+v", in.Mapper)
	err := saveMapper(in.Mapper)
	if err != nil {
		klog.Errorf("fail to save mapper %s to db with err: %v", in.Mapper.Name, err)
		return nil, err
	}
	s.mapperList[in.Mapper.Name] = in.Mapper
	if !in.WithData {
		return &pb.MapperRegisterResponse{}, nil
	}

	var deviceList []*pb.Device
	var deviceModelList []*pb.DeviceModel
	for _, device := range s.deviceList {
		protocol, err := dtcommon.GetProtocolNameOfDevice(device)

		if err == nil {
			if protocol == in.Mapper.Protocol {
				dev, err := dtcommon.ConvertDevice(device)
				if err != nil {
					klog.Errorf("fail to convert device %s with err: %v", device.Name, err)
					continue
				}
				dm, err := dtcommon.ConvertDeviceModel(s.deviceModelList[device.Spec.DeviceModelRef.Name])
				if err != nil {
					klog.Errorf("fail to convert device model %s with err: %v", s.deviceModelList[device.Spec.DeviceModelRef.Name], err)
					continue
				}
				deviceList = append(deviceList, dev)
				deviceModelList = append(deviceModelList, dm)
			}
		}
	}

	return &pb.MapperRegisterResponse{
		DeviceList: deviceList,
		ModelList:  deviceModelList,
	}, nil
}

func (s *server) ReportDeviceStatus(ctx context.Context, in *pb.ReportDeviceStatusRequest) (*pb.ReportDeviceStatusResponse, error) {
	for _, twin := range in.ReportedDevice.Twins {
		propertyType, ok := twin.Reported.Metadata[PropertyType]
		if !ok {
			errLog := fmt.Sprintf("fail to get propertyType for property %s of device %s", twin.PropertyName, in.DeviceName)
			klog.Errorf(errLog)
			return nil, fmt.Errorf(errLog)
		}
		msg, err := CreateMessageTwinUpdate(twin.PropertyName, propertyType, twin.Reported.Value)
		if err != nil {
			klog.Errorf("fail to create message data for property %s of device %s with err: %v", twin.PropertyName, in.DeviceName, err)
			return nil, err
		}
		handleDeviceTwin(in.DeviceName, msg)
	}

	return &pb.ReportDeviceStatusResponse{}, nil
}

func handleDeviceTwin(deviceName string, payload []byte) {
	topic := dtcommon.DeviceETPrefix + deviceName + dtcommon.TwinETUpdateSuffix
	target := modules.TwinGroup
	resource := base64.URLEncoding.EncodeToString([]byte(topic))
	// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
	message := beehiveModel.NewMessage("").BuildRouter(modules.BusGroup, modules.UserGroup,
		resource, messagepkg.OperationResponse).FillBody(string(payload))

	beehiveContext.SendToGroup(target, *message)
}

// CreateMessageTwinUpdate create twin update message.
func CreateMessageTwinUpdate(name string, valueType string, value string) (msg []byte, err error) {
	var updateMsg DeviceTwinUpdate

	updateMsg.BaseMessage.Timestamp = getTimestamp()
	updateMsg.Twin = map[string]*MsgTwin{}
	updateMsg.Twin[name] = &MsgTwin{}
	updateMsg.Twin[name].Actual = &TwinValue{Value: &value}
	updateMsg.Twin[name].Metadata = &TypeMetadata{Type: valueType}

	msg, err = json.Marshal(updateMsg)
	return
}

func initSock(sockPath string) error {
	klog.Infof("init uds socket: %s", sockPath)
	_, err := os.Stat(sockPath)
	if err == nil {
		err = os.Remove(sockPath)
		if err != nil {
			return err
		}
		return nil
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return fmt.Errorf("fail to stat uds socket path")
	}
}

func StartDMIServer(mapperList map[string]*pb.MapperInfo, deviceList map[string]*v1alpha2.Device, deviceModelList map[string]*v1alpha2.DeviceModel) {
	err := initSock(SockPath)
	if err != nil {
		klog.Fatalf("failed to remove uds socket with err: %v", err)
		return
	}

	lis, err := net.Listen(UnixNetworkType, SockPath)
	if err != nil {
		klog.Errorf("failed to start DMI Server with err: %v", err)
		return
	}

	s := grpc.NewServer()
	pb.RegisterDeviceManagerServiceServer(s, &server{
		mapperList:      mapperList,
		deviceList:      deviceList,
		deviceModelList: deviceModelList,
	})
	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		klog.Errorf("failed to start DMI Server with err: %v", err)
	}
	klog.Infoln("success to start DMI Server")
}

func saveMapper(mapper *pb.MapperInfo) error {
	content, err := json.Marshal(mapper)
	if err != nil {
		klog.Errorf("marshal mapper info failed, %s: %v", mapper.Name, err)
		return err
	}
	resource := fmt.Sprintf("%s%s%s%s%s%s%s%s%s", "node", constants.ResourceSep, "nodeID",
		constants.ResourceSep, "namespace", constants.ResourceSep, ResourceTypeDeviceMapper, constants.ResourceSep, mapper.Name)
	meta := &dao.Meta{
		Key:   resource,
		Type:  ResourceTypeDeviceMapper,
		Value: string(content)}
	err = dao.SaveMeta(meta)
	if err != nil {
		klog.Errorf("save meta failed, %s: %v", mapper.Name, err)
		return err
	}
	klog.Infof("success to save mapper info of %s to db", mapper.Name)
	return nil
}
