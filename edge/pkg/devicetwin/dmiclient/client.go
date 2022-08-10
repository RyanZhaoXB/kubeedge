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

func GenerateDMIClient(sockPath string) (dmiapi.DeviceMapperServiceClient, context.Context, *grpc.ClientConn, context.CancelFunc, error) {
	dialer := func(addr string, t time.Duration) (net.Conn, error) {
		return net.Dial(deviceconst.UnixNetworkType, addr)
	}

	conn, err := grpc.Dial(sockPath, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		klog.Errorf("did not connect: %v\n", err)
		return nil, nil, nil, nil, err
	}

	c := dmiapi.NewMapperClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	return c, ctx, conn, cancel, nil
}

func CreateDeviceRequest(device *v1alpha2.Device) (*dmiapi.CreateDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}

	return &dmiapi.CreateDeviceRequest{
		Device: d,
	}, nil
}

func RemoveDeviceRequest(deviceName string) (*dmiapi.RemoveDeviceRequest, error) {
	return &dmiapi.RemoveDeviceRequest{
		DeviceName: deviceName,
	}, nil
}

func UpdateDeviceRequest(device *v1alpha2.Device) (*dmiapi.UpdateDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}

	return &dmiapi.UpdateDeviceRequest{
		Device: d,
	}, nil
}

func CreateDeviceModelRequest(model *v1alpha2.DeviceModel) (*dmiapi.CreateDeviceModelRequest, error) {
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}

	return &dmiapi.CreateDeviceModelRequest{
		Model: m,
	}, nil
}

func UpdateDeviceModelRequest(model *v1alpha2.DeviceModel) (*dmiapi.UpdateDeviceModelRequest, error) {
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}

	return &dmiapi.UpdateDeviceModelRequest{
		Model: m,
	}, nil
}

func RemoveDeviceModelRequest(deviceModelName string) (*dmiapi.RemoveDeviceModelRequest, error) {
	return &dmiapi.RemoveDeviceModelRequest{
		ModelName: deviceModelName,
	}, nil
}
