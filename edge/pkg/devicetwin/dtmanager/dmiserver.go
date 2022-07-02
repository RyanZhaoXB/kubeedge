package dtmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	pb "github.com/kubeedge/kubeedge/edge/pkg/apis/dmi/v1"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiserver"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

// ResourceTypeDeviceModel is type for device model distribute to node
const (
	ResourceTypeDeviceModel  = "devicemodel"
	ResourceTypeDevice       = "device"
	ResourceTypeDeviceMapper = "devicemapper"
)

//TwinWorker deal twin event
type DMIWorker struct {
	Worker
	Group string
}

var (
	//dmiActionCallBack map for action to callback
	dmiActionCallBack map[string]CallBack

	MapperList        map[string]*pb.MapperInfo
	DeviceModelList   map[string]*v1alpha2.DeviceModel
	DeviceList        map[string]*v1alpha2.Device
	PendingDeviceList map[string]*v1alpha2.Device
)

//Start worker
func (dw DMIWorker) Start() {
	klog.Infoln("dmi worker start")
	initDMIActionCallBack()
	initDeviceModelInfoFromDB()
	initDeviceInfoFromDB()
	initDeviceMapperInfoFromDB()
	initPendingDeviceList()

	go dmiserver.StartDMIServer(MapperList, DeviceList, DeviceModelList)
	go dealPendingDeviceList()

	for {
		select {
		case msg, ok := <-dw.ReceiverChan:
			if !ok {
				return
			}

			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := dmiActionCallBack[dtMsg.Action]; exist {
					err := fn(dw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						klog.Errorf("DMIModule deal %s event failed: %v", dtMsg.Action, err)
					}
				} else {
					klog.Errorf("DMIModule deal %s event failed, not found callback", dtMsg.Action)
				}
			}

		case v, ok := <-dw.HeartBeatChan:
			if !ok {
				return
			}
			if err := dw.DTContexts.HeartBeat(dw.Group, v); err != nil {
				return
			}
		}
	}
}

func initDMIActionCallBack() {
	dmiActionCallBack = make(map[string]CallBack)
	dmiActionCallBack[dtcommon.MetaDeviceOperation] = dealMetaDeviceOperation
}

func dealMetaDeviceOperation(context *dtcontext.DTContext, resource string, msg interface{}) error {
	message, ok := msg.(*model.Message)
	if !ok {
		return errors.New("msg not Message type")
	}
	resources := strings.Split(message.Router.Resource, "/")
	if len(resources) != 3 {
		return fmt.Errorf("wrong resources %s", message.Router.Resource)
	}
	var device v1alpha2.Device
	var dm v1alpha2.DeviceModel
	switch resources[1] {
	case "device":
		switch message.GetOperation() {
		case "insert":
			err := json.Unmarshal(message.Content.([]byte), &device)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}

			dm, ok := DeviceModelList[device.Spec.DeviceModelRef.Name]
			if !ok {
				PendingDeviceList[device.Name] = &device
				return nil
			}
			err = GrpcCreateDevice(&device, dm)
			if err != nil {
				klog.Errorf("add device %s failed with err: %s", device.Name, err)
				return err
			}
			DeviceList[device.Name] = &device
		case "delete":
			err := json.Unmarshal(message.Content.([]byte), &device)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}

			err = GrpcRemoveDevice(&device)
			if err != nil {
				klog.Errorf("delete device %s failed with err: %s", device.Name, err)
				return err
			}
			delete(PendingDeviceList, device.Name)
			delete(DeviceList, device.Name)
		case "update":
			err := json.Unmarshal(message.Content.([]byte), &device)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}

			dm, ok := DeviceModelList[device.Spec.DeviceModelRef.Name]
			if !ok {
				klog.Errorf("udpate device %s failed, cannot find device model", device.Name)
				return fmt.Errorf("udpate device %s failed, cannot find device model", device.Name)
			}
			err = GrpcUpdateDevice(&device, dm)
			if err != nil {
				klog.Errorf("udpate device %s failed with err: %s", device.Name, err)
				return err
			}
			DeviceList[device.Name] = &device
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}
	case "devicemodel":
		switch message.GetOperation() {
		case "insert":
			err := json.Unmarshal(message.Content.([]byte), &dm)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}
			DeviceModelList[dm.Name] = &dm
		case "delete":
			err := json.Unmarshal(message.Content.([]byte), &dm)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}
			delete(DeviceModelList, dm.Name)
		case "update":
			err := json.Unmarshal(message.Content.([]byte), &dm)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}
			DeviceModelList[dm.Name] = &dm
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}

	default:
		klog.Warningf("unsupported resource type %s", resources[3])
	}

	return nil
}

func GrpcCreateDevice(device *v1alpha2.Device, model *v1alpha2.DeviceModel) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}
	sockPath := GetSockPath(protocol)
	if sockPath == "" {
		return fmt.Errorf("cannot get sockPath of protocol %s", protocol)
	}

	dc, ctx, conn, cancelFunc, err := dmiclient.GenerateDMIClient(sockPath)
	if err != nil {
		return err
	}

	defer conn.Close()
	defer cancelFunc()

	cdr, err := dmiclient.CreateDeviceRequest(device, model)
	if err != nil {
		return fmt.Errorf("fail to create CDRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.CreateDevice(ctx, cdr)
	if err != nil {
		return err
	}
	return nil
}

func GrpcRemoveDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}
	sockPath := GetSockPath(protocol)
	if sockPath == "" {
		return fmt.Errorf("cannot get sockPath of protocol %s", protocol)
	}

	dc, ctx, conn, cancelFunc, err := dmiclient.GenerateDMIClient(sockPath)
	if err != nil {
		return err
	}

	defer conn.Close()
	defer cancelFunc()

	cdr, err := dmiclient.RemoveDeviceRequest(device.Name)
	if err != nil {
		return err
	}
	_, err = dc.RemoveDevice(ctx, cdr)
	if err != nil {
		return err
	}
	return nil
}

func GrpcUpdateDevice(device *v1alpha2.Device, model *v1alpha2.DeviceModel) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}
	sockPath := GetSockPath(protocol)
	if sockPath == "" {
		return fmt.Errorf("cannot get sockPath of protocol %s", protocol)
	}

	dc, ctx, conn, cancelFunc, err := dmiclient.GenerateDMIClient(sockPath)
	if err != nil {
		return err
	}

	defer conn.Close()
	defer cancelFunc()

	cdr, err := dmiclient.UpdateDeviceRequest(device, model)
	if err != nil {
		return fmt.Errorf("fail to generate UpdateDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.UpdateDevice(ctx, cdr)
	if err != nil {
		return err
	}
	return nil
}

func GetSockPath(protocol string) string {
	for _, mapper := range MapperList {
		if mapper.GetProtocol() == protocol {
			return string(mapper.Address)
		}
	}
	return ""
}

func initDeviceModelInfoFromDB() {
	DeviceModelList = make(map[string]*v1alpha2.DeviceModel)
	metas, err := dao.QueryMeta("type", ResourceTypeDeviceModel)
	if err != nil {
		klog.Errorf("fail to init device model info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		deviceModel := &v1alpha2.DeviceModel{}
		if err := json.Unmarshal([]byte(meta), &deviceModel); err != nil {
			klog.Errorf("fail to unmarshal device model info from db with err: %v", err)
			return
		}
		klog.Infof("deviceModel: %+v", deviceModel)
		DeviceModelList[deviceModel.Name] = deviceModel
	}
	klog.Infoln("success to init device model info from db")
}

func initPendingDeviceList() {
	PendingDeviceList = make(map[string]*v1alpha2.Device)
}

func initDeviceInfoFromDB() {
	DeviceList = make(map[string]*v1alpha2.Device)
	metas, err := dao.QueryMeta("type", ResourceTypeDevice)
	if err != nil {
		klog.Errorf("fail to init device info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		device := &v1alpha2.Device{}
		if err := json.Unmarshal([]byte(meta), &device); err != nil {
			klog.Errorf("fail to unmarshal device info from db with err: %v", err)
			return
		}
		DeviceList[device.Name] = device
	}
	klog.Infoln("success to init device info from db")
}

func initDeviceMapperInfoFromDB() {
	MapperList = make(map[string]*pb.MapperInfo)
	metas, err := dao.QueryMeta("type", ResourceTypeDeviceMapper)
	if err != nil {
		klog.Errorf("fail to init device mapper info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		deviceMapper := &pb.MapperInfo{}
		if err := json.Unmarshal([]byte(meta), &deviceMapper); err != nil {
			klog.Errorf("fail to unmarshal device mapper info from db with err: %v", err)
			return
		}
		MapperList[deviceMapper.Name] = deviceMapper
	}
	klog.Infoln("success to init device mapper info from db")
}

func dealPendingDeviceList() {
	for {
		for _, device := range PendingDeviceList {
			dm, ok := DeviceModelList[device.Spec.DeviceModelRef.Name]
			if !ok {
				continue
			}
			err := GrpcCreateDevice(device, dm)
			if err != nil {
				klog.Errorf("add device %s failed with err: %s", device.Name, err)
				continue
			}

			DeviceList[device.Name] = device
			delete(PendingDeviceList, device.Name)
		}
	}
}
