/*
Copyright 2022 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dtmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	pb "github.com/kubeedge/kubeedge/edge/pkg/apis/dmi/v1"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiserver"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

//TwinWorker deal twin event
type DMIWorker struct {
	Worker
	Group string
}

var (
	//dmiActionCallBack map for action to callback
	dmiActionCallBack map[string]CallBack

	mapperList      map[string]*pb.MapperInfo
	deviceModelList map[string]*v1alpha2.DeviceModel
	deviceList      map[string]*v1alpha2.Device
)

//Start worker
func (dw DMIWorker) Start() {
	klog.Infoln("dmi worker start")
	initDMIActionCallBack()
	initDeviceModelInfoFromDB()
	initDeviceInfoFromDB()
	initDeviceMapperInfoFromDB()

	go dmiserver.StartDMIServer(mapperList, deviceList, deviceModelList)

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
	case constants.ResourceTypeDevice:
		switch message.GetOperation() {
		case model.InsertOperation:
			err := json.Unmarshal(message.Content.([]byte), &device)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}

			err = GrpcCreateDevice(&device)
			if err != nil {
				klog.Errorf("add device %s failed with err: %s", device.Name, err)
				return err
			}
			deviceList[device.Name] = &device
		case model.DeleteOperation:
			err := json.Unmarshal(message.Content.([]byte), &device)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}

			err = GrpcRemoveDevice(&device)
			if err != nil {
				klog.Errorf("delete device %s failed with err: %s", device.Name, err)
				return err
			}
			delete(deviceList, device.Name)
		case model.UpdateOperation:
			err := json.Unmarshal(message.Content.([]byte), &device)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}

			err = GrpcUpdateDevice(&device)
			if err != nil {
				klog.Errorf("udpate device %s failed with err: %s", device.Name, err)
				return err
			}
			deviceList[device.Name] = &device
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}
	case constants.ResourceTypeDeviceModel:
		switch message.GetOperation() {
		case model.InsertOperation:
			err := json.Unmarshal(message.Content.([]byte), &dm)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}
			err = GrpcCreateDeviceModel(&dm)
			if err != nil {
				klog.Errorf("add device model %s failed with err: %s", dm.Name, err)
				return err
			}

			deviceModelList[dm.Name] = &dm
		case model.DeleteOperation:
			err := json.Unmarshal(message.Content.([]byte), &dm)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}
			err = GrpcRemoveDeviceModel(&dm)
			if err != nil {
				klog.Errorf("delete device model %s failed with err: %s", dm.Name, err)
				return err
			}

			delete(deviceModelList, dm.Name)
		case model.UpdateOperation:
			err := json.Unmarshal(message.Content.([]byte), &dm)
			if err != nil {
				return fmt.Errorf("invalid message content with err: %+v", err)
			}
			err = GrpcUpdateDeviceModel(&dm)
			if err != nil {
				klog.Errorf("update device model %s failed with err: %s", dm.Name, err)
				return err
			}

			deviceModelList[dm.Name] = &dm
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}

	default:
		klog.Warningf("unsupported resource type %s", resources[3])
	}

	return nil
}

func GrpcCreateDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}
	sockPath := GetSockPath(protocol)
	if sockPath == "" {
		return fmt.Errorf("cannot get sockPath of protocol %s", protocol)
	}

	dc, err := dmiserver.GetDMIClientByProtocol(protocol)
	if err != nil {
		return err
	}

	cdr, err := dmiclient.CreateDeviceRequest(device)
	if err != nil {
		return fmt.Errorf("fail to create CreateDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.Client.CreateDevice(dc.Ctx, cdr)
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

	rdr, err := dmiclient.RemoveDeviceRequest(device.Name)
	if err != nil {
		return fmt.Errorf("fail to generate RemoveDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.RemoveDevice(ctx, rdr)
	if err != nil {
		return err
	}
	return nil
}

func GrpcUpdateDevice(device *v1alpha2.Device) error {
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

	udr, err := dmiclient.UpdateDeviceRequest(device)
	if err != nil {
		return fmt.Errorf("fail to generate UpdateDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.UpdateDevice(ctx, udr)
	if err != nil {
		return err
	}
	return nil
}

func GrpcCreateDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
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

	cdmr, err := dmiclient.CreateDeviceModelRequest(model)
	if err != nil {
		return fmt.Errorf("fail to create CreateDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.CreateDeviceModel(ctx, cdmr)
	if err != nil {
		return err
	}
	return nil
}

func GrpcRemoveDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
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

	rdmr, err := dmiclient.RemoveDeviceModelRequest(model.Name)
	if err != nil {
		return fmt.Errorf("fail to create RemoveDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.RemoveDeviceModel(ctx, rdmr)
	if err != nil {
		return err
	}
	return nil
}

func GrpcUpdateDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
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

	udmr, err := dmiclient.UpdateDeviceModelRequest(model)
	if err != nil {
		return fmt.Errorf("fail to create UpdateDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.UpdateDeviceModel(ctx, udmr)
	if err != nil {
		return err
	}
	return nil
}

func GetSockPath(protocol string) string {
	for _, mapper := range mapperList {
		if mapper.GetProtocol() == protocol {
			return string(mapper.Address)
		}
	}
	return ""
}

func initDeviceModelInfoFromDB() {
	deviceModelList = make(map[string]*v1alpha2.DeviceModel)
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDeviceModel)
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
		deviceModelList[deviceModel.Name] = deviceModel
	}
	klog.Infoln("success to init device model info from db")
}

func initDeviceInfoFromDB() {
	deviceList = make(map[string]*v1alpha2.Device)
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDevice)
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
		deviceList[device.Name] = device
	}
	klog.Infoln("success to init device info from db")
}

func initDeviceMapperInfoFromDB() {
	mapperList = make(map[string]*pb.MapperInfo)
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDeviceMapper)
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
		mapperList[deviceMapper.Name] = deviceMapper
	}
	klog.Infoln("success to init device mapper info from db")
}
