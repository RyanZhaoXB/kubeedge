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

package dmiclient

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"

	deviceconst "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/apis/dmi/v1"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

type DmiClient struct {
	Client     dmiapi.DeviceMapperServiceClient
	Ctx        context.Context
	Conn       *grpc.ClientConn
	CancelFunc context.CancelFunc
}

var dmiClients map[string]*DmiClient

func (dc *DmiClient) Close() {
	dc.Conn.Close()
	dc.CancelFunc()
}

func init() {
	dmiClients = make(map[string]*DmiClient)
}

func generateDMIClient(sockPath string) (*DmiClient, error) {
	dialer := func(addr string, t time.Duration) (net.Conn, error) {
		return net.Dial(deviceconst.UnixNetworkType, addr)
	}

	conn, err := grpc.Dial(sockPath, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		klog.Errorf("did not connect: %v\n", err)
		return nil, err
	}

	c := dmiapi.NewMapperClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	return &DmiClient{
		Client:     c,
		Ctx:        ctx,
		Conn:       conn,
		CancelFunc: cancel,
	}, nil
}

func createDeviceRequest(device *v1alpha2.Device) (*dmiapi.CreateDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}

	return &dmiapi.CreateDeviceRequest{
		Device: d,
	}, nil
}

func removeDeviceRequest(deviceName string) (*dmiapi.RemoveDeviceRequest, error) {
	return &dmiapi.RemoveDeviceRequest{
		DeviceName: deviceName,
	}, nil
}

func updateDeviceRequest(device *v1alpha2.Device) (*dmiapi.UpdateDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}

	return &dmiapi.UpdateDeviceRequest{
		Device: d,
	}, nil
}

func createDeviceModelRequest(model *v1alpha2.DeviceModel) (*dmiapi.CreateDeviceModelRequest, error) {
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}

	return &dmiapi.CreateDeviceModelRequest{
		Model: m,
	}, nil
}

func updateDeviceModelRequest(model *v1alpha2.DeviceModel) (*dmiapi.UpdateDeviceModelRequest, error) {
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}

	return &dmiapi.UpdateDeviceModelRequest{
		Model: m,
	}, nil
}

func removeDeviceModelRequest(deviceModelName string) (*dmiapi.RemoveDeviceModelRequest, error) {
	return &dmiapi.RemoveDeviceModelRequest{
		ModelName: deviceModelName,
	}, nil
}

func getDMIClientByProtocol(protocol string) (*DmiClient, error) {
	dc, ok := dmiClients[protocol]
	if !ok {
		return nil, fmt.Errorf("fail to get dmi client of protocol %s", protocol)
	}
	return dc, nil
}

func CreateDMIClientByProtocol(protocol, sockPath string) error {
	client, err := getDMIClientByProtocol(protocol)
	if err == nil {
		client.Close()
	}

	dc, err := generateDMIClient(sockPath)
	if err != nil {
		return err
	}
	dmiClients[protocol] = dc
	return nil
}

func CreateDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}

	dc, err := getDMIClientByProtocol(protocol)
	if err != nil {
		return err
	}

	cdr, err := createDeviceRequest(device)
	if err != nil {
		return fmt.Errorf("fail to create createDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.Client.CreateDevice(dc.Ctx, cdr)
	if err != nil {
		return err
	}
	return nil
}

func RemoveDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}

	dc, err := getDMIClientByProtocol(protocol)
	if err != nil {
		return err
	}

	rdr, err := removeDeviceRequest(device.Name)
	if err != nil {
		return fmt.Errorf("fail to generate RemoveDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.Client.RemoveDevice(dc.Ctx, rdr)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}

	dc, err := getDMIClientByProtocol(protocol)
	if err != nil {
		return err
	}

	udr, err := updateDeviceRequest(device)
	if err != nil {
		return fmt.Errorf("fail to generate UpdateDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.Client.UpdateDevice(dc.Ctx, udr)
	if err != nil {
		return err
	}
	return nil
}

func CreateDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
	dc, err := getDMIClientByProtocol(protocol)
	if err != nil {
		return err
	}

	cdmr, err := createDeviceModelRequest(model)
	if err != nil {
		return fmt.Errorf("fail to create CreateDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.Client.CreateDeviceModel(dc.Ctx, cdmr)
	if err != nil {
		return err
	}
	return nil
}

func RemoveDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
	dc, err := getDMIClientByProtocol(protocol)
	if err != nil {
		return err
	}

	rdmr, err := removeDeviceModelRequest(model.Name)
	if err != nil {
		return fmt.Errorf("fail to create RemoveDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.Client.RemoveDeviceModel(dc.Ctx, rdmr)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
	dc, err := getDMIClientByProtocol(protocol)
	if err != nil {
		return err
	}

	udmr, err := updateDeviceModelRequest(model)
	if err != nil {
		return fmt.Errorf("fail to create UpdateDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.Client.UpdateDeviceModel(dc.Ctx, udmr)
	if err != nil {
		return err
	}
	return nil
}
