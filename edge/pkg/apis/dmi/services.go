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

package dmi

import (
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/apis/dmi/v1"
)

// DeviceManagerService defines the public APIS for remote device management.
// The server is implemented by the module of device manager in edgecore
// and the client is implemented by the device mapper for upstreaming.
type DeviceManagerService interface {
	// MapperRegister register the information of the mapper to device manager.
	MapperRegister(*dmiapi.MapperRegisterRequest) (*dmiapi.MapperRegisterResponse, error)
	// ReportDeviceStatus report the status of devices to device manager.
	ReportDeviceStatus(*dmiapi.ReportDeviceStatusRequest) (*dmiapi.ReportDeviceStatusResponse, error)
}

// DeviceMapperService defines the public APIS for remote device management.
// The server is implemented by the device mapper
// and the client is implemented by the module of device manager in edgecore for downstreaming.
type DeviceMapperService interface {
	// CreateDevice register a device to the device mapper
	// and the mapper will connect to the physical device with the device information.
	CreateDevice(*dmiapi.CreateDeviceRequest) (*dmiapi.CreateDeviceResponse, error)
	// RemoveDevice unregister a device to the device mapper
	// and the mapper will disconnect to the physical device by name.
	RemoveDevice(*dmiapi.RemoveDeviceRequest) (*dmiapi.RemoveDeviceResponse, error)
	// UpdateDevice update the specification information of a device in the device mapper.
	UpdateDevice(*dmiapi.UpdateDeviceRequest) (*dmiapi.UpdateDeviceResponse, error)
	// UpdateDeviceStatus update the status including device twin and state of a device status
	// to the device mapper and the mapper will change the actual status of the physical device.
	UpdateDeviceStatus(*dmiapi.UpdateDeviceStatusRequest) (*dmiapi.UpdateDeviceStatusResponse, error)
	// GetDevice get the information of a device from the device mapper.
	GetDevice(*dmiapi.GetDeviceRequest) (*dmiapi.GetDeviceResponse, error)

	// CreateDeviceModel create a device model to the device mapper.
	CreateDeviceModel(*dmiapi.CreateDeviceRequest) (*dmiapi.CreateDeviceResponse, error)
	// RemoveDeviceModel remove a device model to the device mapper.
	RemoveDeviceModel(*dmiapi.RemoveDeviceRequest) (*dmiapi.RemoveDeviceResponse, error)
	// UpdateDeviceModel update a device model to the device mapper.
	UpdateDeviceModel(*dmiapi.UpdateDeviceRequest) (*dmiapi.UpdateDeviceResponse, error)
}
