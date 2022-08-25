package dtcommon

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	pb "github.com/kubeedge/kubeedge/edge/pkg/apis/dmi/v1"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

//ValidateValue validate value type
func ValidateValue(valueType string, value string) error {
	switch valueType {
	case "":
		valueType = "string"
		return nil
	case "string":
		return nil
	case "int", "integer":
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("the value is not int or integer")
		}
		return nil
	case "float":
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("the value is not float")
		}
		return nil
	case "boolean":
		if strings.Compare(value, "true") != 0 && strings.Compare(value, "false") != 0 {
			return errors.New("the bool value must be true or false")
		}
		return nil
	case TypeDeleted:
		return nil
	default:
		return errors.New("the value type is not allowed")
	}
}

//ValidateTwinKey validate twin key
func ValidateTwinKey(key string) bool {
	pattern := "^[a-zA-Z0-9-_.,:/@#]{1,128}$"
	match, _ := regexp.MatchString(pattern, key)
	return match
}

//ValidateTwinValue validate twin value
func ValidateTwinValue(value string) bool {
	pattern := "^[a-zA-Z0-9-_.,:/@#]{1,512}$"
	match, _ := regexp.MatchString(pattern, value)
	return match
}

func GetProtocolNameOfDevice(device *v1alpha2.Device) (string, error) {
	protocol := device.Spec.Protocol
	if protocol.OpcUA != nil {
		return "OPCUA", nil
	}
	if protocol.Modbus != nil {
		return "Modbus", nil
	}
	if protocol.Bluetooth != nil {
		return "Bluetooth", nil
	}
	if protocol.CustomizedProtocol != nil {
		return protocol.CustomizedProtocol.ProtocolName, nil
	}
	return "", fmt.Errorf("cannot find protocol name for device %s", device.Name)
}
func ConvertDevice(device *v1alpha2.Device) (*pb.Device, error) {
	data, err := json.Marshal(device)
	if err != nil {
		klog.Errorf("fail to marshal device %s with err: %v", device.Name, err)
		return nil, err
	}

	var edgeDevice pb.Device
	err = json.Unmarshal(data, &edgeDevice)
	if err != nil {
		klog.Errorf("fail to unmarshal device %s with err: %v", device.Name, err)
		return nil, err
	}

	edgeDevice.Name = device.Name
	edgeDevice.Spec.DeviceModelReference = device.Spec.DeviceModelRef.Name

	return &edgeDevice, nil
}

func ConvertDeviceModel(model *v1alpha2.DeviceModel) (*pb.DeviceModel, error) {
	data, err := json.Marshal(model)
	if err != nil {
		klog.Errorf("fail to marshal device model %s with err: %v", model.Name, err)
		return nil, err
	}

	var edgeDeviceModel pb.DeviceModel
	err = json.Unmarshal(data, &edgeDeviceModel)
	if err != nil {
		klog.Errorf("fail to unmarshal device model %s with err: %v", model.Name, err)
		return nil, err
	}
	edgeDeviceModel.Name = model.Name

	return &edgeDeviceModel, nil
}
