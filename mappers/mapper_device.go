package mappers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"

	"github.com/kubeedge/kubeedge/common/constants"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	dmiapi "github.com/kubeedge/kubeedge/edge/pkg/dmi/apis/v1"
	"github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/dtclient"
	"github.com/kubeedge/kubeedge/mappers/dtcommon"
	"github.com/kubeedge/kubeedge/mappers/dttype"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

func (d *DeviceTwinShim) CreateDevice(config *dmiapi.DeviceConfig) (string, error) {
	klog.V(2).Info("CreateDevice")
	if config == nil {
		return "", errors.New("device config is nil")
	}
	device := config.Device
	model := config.Model
	if _, exists := d.getDeviceInstance(device.Name); exists {
		return "", fmt.Errorf("add device %s failed, has existed", device.Name)
	}

	// filter node
	if device.Spec.NodeSelector == nil ||
		len(device.Spec.NodeSelector.NodeSelectorTerms) == 0 ||
		len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions) == 0 ||
		len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values) == 0 {
		return "", fmt.Errorf("add device %s failed, device has not specify node", device.Name)
	}
	isNodeFilter := false
	for _, value := range device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values {
		if value == d.NodeName {
			isNodeFilter = true
			break
		}
	}
	if !isNodeFilter {
		return "", fmt.Errorf("add device %s failed, device nodeSelector value is different to current node", device.Name)
	}

	// store in cache
	typeDevice := &dttype.Device{
		ID:          device.Name,
		Name:        device.Name,
		Description: device.Labels["description"],
		// TODO use constants from mapper-go
		State: "unknown",
	}
	if device.Status.Twins != nil {
		typeDevice.Twin = make(map[string]*dttype.MsgTwin)
	}
	if model.Spec.Properties != nil {
		typeDevice.Attributes = make(map[string]*dttype.MsgAttr)
	}
	d.deviceMutex.Store(device.Name, &sync.Mutex{})
	d.deviceList.Store(device.Name, typeDevice)

	// write to sqlite
	var err error
	// dtclient device have not attrs and twins
	protocolData, _ := json.Marshal(device.Spec.Protocol)
	saveDevice := dtclient.Device{
		ID:          device.Name,
		Name:        device.Name,
		Description: device.Labels["description"],
		// TODO use constants from mapper-go
		State:      "unknown",
		LastOnline: "",
		Protocol:   string(protocolData),
	}
	for i := 1; i <= common.RetryTimes; i++ {
		if err = dtclient.AddDeviceTrans([]dtclient.Device{saveDevice}, nil, nil); err == nil {
			break
		}
		time.Sleep(common.RetryInterval)
	}

	if err != nil {
		d.deviceList.Delete(device.Name)
		d.deviceMutex.Delete(device.Name)
		return "", fmt.Errorf("add device %s failed due to some error, err: %#v", device.Name, err)
	}

	// save twin to sqlite
	if device.Status.Twins != nil {
		klog.V(2).Infof("Add device twin during first adding device %s", device.Name)
		twins := buildMsgTwins(device.Status.Twins, device.ResourceVersion, false)
		addTwins := make([]dtclient.DeviceTwin, 0, len(twins))
		if err = dealTwinAdd(&addTwins, device.Name, twins); err != nil {
			return "", fmt.Errorf("add device deal twin failed, device: %s, err: %v", device.Name, err)
		}
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			if err = dtclient.DeviceTwinTrans(addTwins, nil, nil); err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		if err != nil {
			d.syncDeviceFromSqlite(device.Name)
			return "", fmt.Errorf("add device twin failed due to writing sql error: %v", err)
		}
		typeDevice.Twin = twins
		// TODO send twin to mqtt? document, delta, syncResult?
	}

	// save attr to sqlite
	if model.Spec.Properties != nil {
		klog.V(2).Infof("Add device attr during first adding device %s", device.Name)
		attrs := buildMsgAttrs(model)
		addAttrs := make([]dtclient.DeviceAttr, 0, 0)
		if err = dealAttrAdd(&addAttrs, device.Name, attrs); err != nil {
			return "", fmt.Errorf("add device deal attr failed, device: %s, err: %v", device.Name, err)
		}
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			if err = dtclient.DeviceAttrTrans(addAttrs, nil, nil); err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		if err != nil {
			d.syncDeviceFromSqlite(device.Name)
			return "", fmt.Errorf("add device attr failed due to writing sql error: %v", err)
		}
		typeDevice.Attributes = attrs

		klog.Infof("send update attributes of device %s event to edge app", device.Name)
		baseMessage := dttype.BuildBaseMessage()
		payload, err := dttype.BuildDeviceAttrUpdate(baseMessage, attrs)
		if err != nil {
			klog.Errorf("CreateDevice build device attribute update failed: %v", err)
		}
		topic := dtcommon.DeviceETPrefix + device.Name + dtcommon.DeviceETUpdatedSuffix
		d.send(device.Name,
			dtcommon.SendToEdge,
			dtcommon.CommModule,
			d.buildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, payload))
	}

	// because the twin and attr of the device are updated, saved this device again
	d.deviceList.Store(device.Name, typeDevice)

	// TODO save device to meta

	// publish device to mqtt
	topic := dtcommon.MemETPrefix + d.NodeName + dtcommon.MemETUpdateSuffix
	baseMessage := dttype.BuildBaseMessage()
	addDeviceResult := dttype.MembershipUpdate{BaseMessage: baseMessage, AddDevices: []dttype.Device{*typeDevice}}
	result, err := dttype.MarshalMembershipUpdate(addDeviceResult)
	if err != nil {
		return "", fmt.Errorf("add device %s failed, marshal membership err: %s", device.Name, err)
	}
	d.send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		d.buildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
	return device.Name, nil
}

func (d *DeviceTwinShim) UpdateDevice(deviceID string, config *dmiapi.DeviceConfig) error {
	klog.V(2).Info("UpdateDevice")
	var err error
	srcDeviceInstance, exists := d.getDeviceInstance(deviceID)
	if !exists {
		return fmt.Errorf("update device %s failed, not exists", deviceID)
	}

	device := config.Device
	model := config.Model

	// filter node
	if device.Spec.NodeSelector == nil ||
		len(device.Spec.NodeSelector.NodeSelectorTerms) == 0 ||
		len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions) == 0 ||
		len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values) == 0 {
		return fmt.Errorf("update device %s failed, device has not specify node", device.Name)
	}
	isNodeFilter := false
	for _, value := range device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values {
		if value == d.NodeName {
			isNodeFilter = true
			break
		}
	}
	if !isNodeFilter {
		return fmt.Errorf("update device %s failed, device nodeSelector value is different to current node", device.Name)
	}

	// get device by id from sqlite
	devices, err := dtclient.QueryDevice("id", deviceID)
	if err != nil {
		klog.Errorf("query device failed: %v", err)
		return err
	}
	if len(*devices) <= 0 {
		return fmt.Errorf("not found device %s from db", deviceID)
	}
	srcDevice := (*devices)[0]

	srcDeviceUpdateMap := make(map[string]interface{})
	if device.Name != srcDeviceInstance.Name {
		klog.Warningf("update device name is unsupported, continue. from %s to %s", deviceID, device.Name)
	}
	// deal description
	srcDeviceInstance.Description = device.Labels["description"]
	srcDeviceUpdateMap["description"] = device.Labels["description"]

	// deal protocol
	protocolData, _ := json.Marshal(device.Spec.Protocol)
	if string(protocolData) != srcDevice.Protocol {
		srcDeviceUpdateMap["protocol"] = string(protocolData)
	}

	// update srcDevice to sqlite
	if err = dtclient.UpdateDeviceFields(deviceID, srcDeviceUpdateMap); err != nil {
		d.syncDeviceFromSqlite(deviceID)
		return fmt.Errorf("update device failed due to writing sql error: %v", err)
	}

	// TODO update device from meta

	// deal attr
	// merge srcDeviceInstance and model attr by dealAttrUpdate
	attrs := buildMsgAttrs(model)
	srcAttrs := srcDeviceInstance.Attributes
	dealAttrResult := dealAttrUpdate(deviceID, srcAttrs, attrs)
	if dealAttrResult.Err != nil {
		return err
	}

	// save cache
	srcDeviceInstance.Attributes = srcAttrs
	d.deviceList.Store(deviceID, srcDeviceInstance)

	// save attr to sqlite and publish attrs to mqtt
	add, deletes, update, result := dealAttrResult.Add, dealAttrResult.Delete, dealAttrResult.Update, dealAttrResult.Result
	if len(add) != 0 || len(deletes) != 0 || len(update) != 0 {
		baseMessage := dttype.BuildBaseMessage()
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			err = dtclient.DeviceAttrTrans(add, deletes, update)
			if err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		baseMessage.Timestamp = time.Now().UnixNano() / 1e6

		if err != nil {
			d.syncDeviceFromSqlite(deviceID)
			return fmt.Errorf("update device failed due to writing sql error: %v", err)
		} else {
			klog.Infof("send update attributes of device %s event to edge app", deviceID)
			payload, err := dttype.BuildDeviceAttrUpdate(baseMessage, result)
			if err != nil {
				return fmt.Errorf("build device %s attribute update failed: %v", deviceID, err)
			}
			topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.DeviceETUpdatedSuffix
			d.send(deviceID, dtcommon.SendToEdge, dtcommon.CommModule,
				d.buildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, payload))
		}
	}

	return nil
}

func (d *DeviceTwinShim) RemoveDevice(deviceID string, deviceName string) error {
	klog.V(2).Info("RemoveDevice")
	device, exists := d.getDeviceInstance(deviceID)
	if !exists {
		return fmt.Errorf("remove device %s failed, not exists", deviceID)
	}
	// delete from sqlite
	for i := 1; i <= dtcommon.RetryTimes; i++ {
		err := dtclient.DeleteDeviceTrans([]string{device.ID})
		if err != nil {
			klog.Errorf("Delete document of device %s failed at %d time, err: %#v", device.ID, i, err)
		} else {
			klog.Infof("Delete document of device %s successful", device.ID)
			break
		}
		time.Sleep(dtcommon.RetryInterval)
	}
	d.deviceList.Delete(deviceID)
	d.deviceMutex.Delete(deviceID)

	// TODO delete from meta

	// send delete operation to mqtt
	topic := dtcommon.MemETPrefix + d.NodeName + dtcommon.MemETUpdateSuffix
	baseMessage := dttype.BuildBaseMessage()
	deleteResult := dttype.MembershipUpdate{BaseMessage: baseMessage, RemoveDevices: []dttype.Device{*device}}
	result, err := dttype.MarshalMembershipUpdate(deleteResult)
	if err != nil {
		klog.Errorf("Remove device %s failed, marshal membership err: %s", device.ID, err)
	}
	d.send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		d.buildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
	klog.Infof("Remove device %s successful", device.ID)
	return nil
}

func (d *DeviceTwinShim) UpdateDeviceStatus(deviceID string, deviceName string, desiredDevice *v1alpha2.DeviceStatus) error {
	// for the update device status, all desired twin is considered to add
	klog.V(2).Info("UpdateDeviceStatus")
	var err error
	device, exists := d.getDeviceInstance(deviceID)
	if !exists {
		return fmt.Errorf("update device %s status failed, not exists", deviceID)
	}
	if desiredDevice.State != "" && device.State != desiredDevice.State {
		device.State = desiredDevice.State
	}
	// save twin to sqlite
	if desiredDevice.Twins != nil {
		klog.V(2).Infof("update device twin for device %s", device.ID)
		desiredTwinMap := buildMsgTwins(desiredDevice.Twins, "", false)
		addTwins := make([]dtclient.DeviceTwin, 0, len(desiredTwinMap))
		if err = dealTwinAdd(&addTwins, device.ID, desiredTwinMap); err != nil {
			return fmt.Errorf("update device status deal twin failed, device: %s, err: %v", device.ID, err)
		}
		// remove original twin
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			if err = dtclient.DeleteDeviceTwinByDeviceID(device.ID); err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		if err != nil {
			d.syncDeviceFromSqlite(device.ID)
			return fmt.Errorf("update device %s twin failed due to writing sql error: %v", device.ID, err)
		}
		// add desired twin
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			if err = dtclient.DeviceTwinTrans(addTwins, nil, nil); err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		if err != nil {
			d.syncDeviceFromSqlite(device.ID)
			return fmt.Errorf("update device %s twin failed due to writing sql error: %v", device.ID, err)
		}
		device.Twin = desiredTwinMap
	}
	// save to cache
	d.deviceList.Store(deviceID, device)

	// publish device to mqtt
	// TODO if desiredDevice twin different to src twin, such as an extra columnA, missing columnB, update columnC, does result need to add missing columnB?
	topic := dtcommon.MemETPrefix + d.NodeName + dtcommon.MemETUpdateSuffix
	baseMessage := dttype.BuildBaseMessage()
	addDeviceResult := dttype.MembershipUpdate{BaseMessage: baseMessage, AddDevices: []dttype.Device{*device}}
	result, err := dttype.MarshalMembershipUpdate(addDeviceResult)
	if err != nil {
		return fmt.Errorf("update device status %s failed, marshal membership err: %s", device.ID, err)
	}
	d.send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		d.buildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
	return nil
}

func (d *DeviceTwinShim) ReportDeviceStatus(deviceID string, deviceName string, reportedDevice *v1alpha2.DeviceStatus) error {
	klog.V(2).Info("ReportDeviceStatus")
	device, exists := d.getDeviceInstance(deviceID)
	if !exists {
		return fmt.Errorf("report device %s status failed, not exists", deviceID)
	}
	baseMessage := dttype.BuildBaseMessage()
	status := dttype.DeviceStatusResult{
		BaseMessage: baseMessage,
		State:       device.State,
		Twins:       device.Twin,
	}
	if reportedDevice != nil {
		status.State = reportedDevice.State
		status.Twins = buildMsgTwins(reportedDevice.Twins, "", true)
		device.State = status.State
		device.Twin = status.Twins
		d.deviceList.Store(device.ID, device)
	}
	result, err := dttype.MarshalDeviceStatusResult(status)
	if err != nil {
		return fmt.Errorf("report device status %s failed, marshal device status err: %s", device.ID, err)
	}
	d.send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		d.buildModelMessage(constants.ResourceGroupDeviceManager, "", "report_status", messagepkg.OperationResponse, result))
	return nil
}

func (d *DeviceTwinShim) PatchDeviceStatus(deviceID string, deviceName string, desiredDevice *v1alpha2.DeviceStatus) error {
	klog.V(2).Info("PatchDeviceStatus")
	var err error
	device, exists := d.getDeviceInstance(deviceID)
	if !exists {
		return fmt.Errorf("patch device %s status failed, not exists", deviceID)
	}
	if desiredDevice.State != "" {
		device.State = desiredDevice.State
	}
	// save twin to sqlite
	if desiredDevice.Twins != nil {
		klog.V(2).Infof("patch device twin for device %s", device.ID)
		desiredTwinMap := buildMsgTwins(desiredDevice.Twins, "", false)
		twinUpdateResult := dealTwinUpdate(device.ID, device.Twin, desiredTwinMap)
		if twinUpdateResult.Err != nil {
			return fmt.Errorf("patch device status deal twin failed, device: %s, err: %v", device.ID, err)
		}
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			if err = dtclient.DeviceTwinTrans(twinUpdateResult.Add, twinUpdateResult.Delete, twinUpdateResult.Update); err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		if err != nil {
			d.syncDeviceFromSqlite(device.ID)
			return fmt.Errorf("patch device %s twin failed due to writing sql error: %v", device.ID, err)
		}
		device.Twin = desiredTwinMap

		// TODO result, need to send to mqtt
	}
	// save to cache
	d.deviceList.Store(deviceID, device)

	// publish device to mqtt
	// TODO twinUpdateResult result have not set
	topic := dtcommon.MemETPrefix + d.NodeName + dtcommon.MemETUpdateSuffix
	baseMessage := dttype.BuildBaseMessage()
	addDeviceResult := dttype.MembershipUpdate{BaseMessage: baseMessage, AddDevices: []dttype.Device{*device}}
	result, err := dttype.MarshalMembershipUpdate(addDeviceResult)
	if err != nil {
		return fmt.Errorf("patch device status %s failed, marshal membership err: %s", device.ID, err)
	}
	d.send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		d.buildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
	return nil
}

func (d *DeviceTwinShim) ListDevices(filter *dmiapi.DeviceFilter) ([]*v1alpha2.DeviceStatus, error) {
	// TODO maybe implement it? Due to deviceTwin storage struct, it seem that impossible to implement this method
	return nil, fmt.Errorf("device twin shim does not support list devices, continue")
}

func (d *DeviceTwinShim) GetDevice(deviceID string, deviceName string) (*v1alpha2.DeviceStatus, error) {
	klog.V(2).Info("GetDevice")
	var err error
	device, exists := d.getDeviceInstance(deviceID)
	if !exists {
		return nil, fmt.Errorf("get device %s failed, not exists", deviceID)
	}

	// send to mqtt get status
	twinDelta, ok := dttype.BuildDeviceTwinDelta(dttype.BuildBaseMessage(), device.Twin)
	if !ok {
		return nil, fmt.Errorf("get device %s failed, build device twin delta failed", device.ID)
	}
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETDeltaSuffix
	klog.Infof("get device %s: send delta", deviceID)
	d.send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		d.buildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, twinDelta))

	// TODO maybe another way?
	// wait for response
	res := &v1alpha2.DeviceStatus{}
	// TODO configurable
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	select {
	case msg, ok := <-d.CommChan[d.Name()]:
		if !ok {
			return nil, fmt.Errorf("failed to get device %s", device.ID)
		}
		if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
			if dtMsg.Msg == nil {
				return nil, fmt.Errorf("get device %s failed, msg is nil", device.ID)
			}
			if err = json.Unmarshal(dtMsg.Msg.Content.([]byte), &res); err != nil {
				return nil, fmt.Errorf("get device %s failed, unmarshal message content failed, err: %v", device.ID, err)
			}
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("time out to get device %s", device.ID)
	}

	// save new status to sqlite
	msgTwins := buildMsgTwins(res.Twins, "", true)
	twinUpdateResult := dealTwinUpdate(device.ID, device.Twin, msgTwins)
	if twinUpdateResult.Err != nil {
		return nil, fmt.Errorf("get device %s save twin failed, err: %v", device.ID, err)
	}
	for i := 1; i <= dtcommon.RetryTimes; i++ {
		if err = dtclient.DeviceTwinTrans(twinUpdateResult.Add, twinUpdateResult.Delete, twinUpdateResult.Update); err == nil {
			break
		}
		time.Sleep(dtcommon.RetryInterval)
	}
	if err != nil {
		d.syncDeviceFromSqlite(device.ID)
		return nil, fmt.Errorf("add device attr failed due to writing sql error: %v", err)
	}

	// save new status to cache
	device.State = res.State
	device.Twin = msgTwins
	d.deviceList.Store(device.ID, device)

	return res, nil
}

func buildMsgTwins(src []v1alpha2.Twin, resourceVersion string, withActual bool) map[string]*dttype.MsgTwin {
	twins := make(map[string]*dttype.MsgTwin, len(src))
	for _, twin := range src {
		now := time.Now().UnixNano() / 1e6
		expected := &dttype.TwinValue{}
		expected.Value = &twin.Desired.Value
		metadataType, ok := twin.Desired.Metadata["type"]
		if !ok {
			metadataType = constants.DataTypeString
		}

		expected.Metadata = &dttype.ValueMetadata{Timestamp: now}
		msgTwin := &dttype.MsgTwin{
			Expected: expected,
			Optional: pointer.Bool(false),
			Metadata: &dttype.TypeMetadata{Type: metadataType},
		}

		if withActual {
			msgTwin.Actual = &dttype.TwinValue{
				Value:    &twin.Reported.Value,
				Metadata: &dttype.ValueMetadata{Timestamp: now},
			}
		}

		if resourceVersion != "" {
			cloudVersion, err := strconv.ParseInt(resourceVersion, 10, 64)
			if err != nil {
				klog.Warningf("Failed to parse cloud version due to error %v", err)
			}
			msgTwin.ExpectedVersion = &dttype.TwinVersion{CloudVersion: cloudVersion, EdgeVersion: 0}
		}
		twins[twin.PropertyName] = msgTwin
	}
	return twins
}

func dealAddTwinVersion(version *dttype.TwinVersion, reqVersion *dttype.TwinVersion) error {
	if reqVersion == nil {
		// TODO
		return nil
	}
	if version.CloudVersion > reqVersion.CloudVersion {
		return errors.New("version not allowed")
	}
	version.CloudVersion = reqVersion.CloudVersion
	version.EdgeVersion = reqVersion.EdgeVersion
	return nil
}

func dealTwinAdd(result *[]dtclient.DeviceTwin, deviceID string, twins map[string]*dttype.MsgTwin) error {
	for key, twin := range twins {
		now := time.Now().UnixNano() / 1e6
		addTwin := dttype.MsgTwinToDeviceTwin(key, twin)
		addTwin.DeviceID = deviceID
		//add deleted twin when syncing from cloud: add version
		if twin.Metadata.Type == dtcommon.TypeDeleted {
			if twin.ExpectedVersion != nil {
				versionJSON, _ := json.Marshal(twin.ExpectedVersion)
				addTwin.ExpectedVersion = string(versionJSON)
			}
			if twin.ActualVersion != nil {
				versionJSON, _ := json.Marshal(twin.ActualVersion)
				addTwin.ActualVersion = string(versionJSON)
			}
		}

		if twin.Expected != nil {
			version := &dttype.TwinVersion{}
			msgTwinExpectedVersion := twin.ExpectedVersion
			if err := dealAddTwinVersion(version, msgTwinExpectedVersion); err != nil {
				// reject add twin
				return err
			}
			// value type default string
			valueType := constants.DataTypeString
			if twin.Metadata != nil {
				valueType = twin.Metadata.Type
			}

			if err := dtcommon.ValidateValue(valueType, *twin.Expected.Value); err != nil {
				return err
			}
			meta := dttype.ValueMetadata{Timestamp: now}
			metaJSON, _ := json.Marshal(meta)
			versionJSON, _ := json.Marshal(version)
			addTwin.ExpectedMeta = string(metaJSON)
			addTwin.ExpectedVersion = string(versionJSON)
			addTwin.Expected = *twin.Expected.Value
		}

		if twin.Actual != nil {
			version := &dttype.TwinVersion{}
			msgTwinActualVersion := twin.ActualVersion
			if err := dealAddTwinVersion(version, msgTwinActualVersion); err != nil {
				return err
			}
			valueType := constants.DataTypeString
			if twin.Metadata != nil {
				valueType = twin.Metadata.Type
			}
			if err := dtcommon.ValidateValue(valueType, *twin.Actual.Value); err != nil {
				return err
			}
			meta := dttype.ValueMetadata{Timestamp: now}
			metaJSON, _ := json.Marshal(meta)
			versionJSON, _ := json.Marshal(version)
			addTwin.ActualMeta = string(metaJSON)
			addTwin.ActualVersion = string(versionJSON)
			addTwin.Actual = *twin.Actual.Value
		}

		//add the optional of twin
		if twin.Optional != nil {
			optional := *twin.Optional
			addTwin.Optional = optional
		} else {
			addTwin.Optional = true
		}

		//add the metadata of the twin
		if twin.Metadata != nil {
			addTwin.AttrType = twin.Metadata.Type
			twin.Metadata.Type = ""
			metaJSON, _ := json.Marshal(twin.Metadata)
			addTwin.Metadata = string(metaJSON)
			twin.Metadata.Type = addTwin.AttrType
		} else {
			addTwin.AttrType = constants.DataTypeString
		}

		twins[key] = dttype.DeviceTwinToMsgTwin([]dtclient.DeviceTwin{addTwin})[key]
		*result = append(*result, addTwin)
	}
	return nil
}

func buildMsgAttrs(model v1alpha2.DeviceModel) map[string]*dttype.MsgAttr {
	attrs := make(map[string]*dttype.MsgAttr, len(model.Spec.Properties))
	for _, property := range model.Spec.Properties {
		attr := &dttype.MsgAttr{
			Optional: pointer.Bool(false),
		}
		if property.Type.Int != nil {
			attr.Value = strconv.FormatInt(property.Type.Int.DefaultValue, 10)
			attr.Metadata = &dttype.TypeMetadata{Type: constants.DataTypeInt}
		} else if property.Type.String != nil {
			attr.Value = property.Type.String.DefaultValue
			attr.Metadata = &dttype.TypeMetadata{Type: constants.DataTypeString}
		} else if property.Type.Double != nil {
			attr.Value = strconv.FormatFloat(property.Type.Double.DefaultValue, 'f', 2, 64)
			attr.Metadata = &dttype.TypeMetadata{Type: constants.DataTypeDouble}
		} else if property.Type.Float != nil {
			attr.Value = strconv.FormatFloat(float64(property.Type.Float.DefaultValue), 'f', 2, 32)
			attr.Metadata = &dttype.TypeMetadata{Type: constants.DataTypeFloat}
		} else if property.Type.Boolean != nil {
			attr.Value = strconv.FormatBool(property.Type.Boolean.DefaultValue)
			attr.Metadata = &dttype.TypeMetadata{Type: constants.DataTypeBoolean}
		} else if property.Type.Bytes != nil {
			attr.Metadata = &dttype.TypeMetadata{Type: constants.DataTypeBytes}
		}
		attrs[property.Name] = attr
	}
	return attrs
}

// dealAttrAdd add device attributes
func dealAttrAdd(result *[]dtclient.DeviceAttr, deviceID string, attributes map[string]*dttype.MsgAttr) error {
	for key, attr := range attributes {
		deviceAttr := dttype.MsgAttrToDeviceAttr(key, attr)
		deviceAttr.DeviceID = deviceID
		deviceAttr.Value = attr.Value
		if attr.Optional != nil {
			optional := *attr.Optional
			deviceAttr.Optional = optional
		}
		if attr.Metadata != nil {
			deviceAttr.AttrType = attr.Metadata.Type
			attr.Metadata.Type = ""
			metaJSON, _ := json.Marshal(attr.Metadata)
			attr.Metadata.Type = deviceAttr.AttrType
			deviceAttr.Metadata = string(metaJSON)
		}
		*result = append(*result, deviceAttr)
		attributes[key] = attr
	}

	return nil
}

func isTwinValueDiff(srcTwin *dttype.MsgTwin, newTwin *dttype.MsgTwin, dealActual bool) error {
	hasTwin := false
	hasMsgTwin := false
	twinValue := srcTwin.Expected
	msgTwinValue := newTwin.Expected

	if dealActual {
		twinValue = srcTwin.Actual
		msgTwinValue = newTwin.Actual
	}
	if twinValue != nil {
		hasTwin = true
	}
	if msgTwinValue != nil {
		hasMsgTwin = true
	}
	valueType := constants.DataTypeString
	if srcTwin.Metadata.Type == dtcommon.TypeDeleted && newTwin.Metadata != nil {
		valueType = newTwin.Metadata.Type
	} else {
		valueType = srcTwin.Metadata.Type
	}
	if hasMsgTwin && hasTwin {
		return dtcommon.ValidateValue(valueType, *msgTwinValue.Value)
	}
	return nil
}

func dealCompareTwinVersion(version *dttype.TwinVersion, reqVersion *dttype.TwinVersion, dealType string) error {
	if reqVersion == nil {
		if dealType == dtcommon.TypeDeleted {
			return nil
		}
		return errors.New("version not allowed be nil while syncing")
	}
	if version.CloudVersion > reqVersion.CloudVersion {
		return errors.New("version not allowed")
	}
	if version.EdgeVersion > reqVersion.EdgeVersion {
		return errors.New("not allowed to sync due to version conflict")
	}
	version.CloudVersion = reqVersion.CloudVersion
	version.EdgeVersion = reqVersion.EdgeVersion
	return nil
}

func dealTwinCompare(deviceID string, key string, twin *dttype.MsgTwin, msgTwin *dttype.MsgTwin) (*dtclient.DeviceTwinUpdate, error) {
	klog.V(2).Info("dealTwinCompare")
	if msgTwin == nil {
		return nil, nil
	}

	now := time.Now().UnixNano() / 1e6
	cols := make(map[string]interface{})
	meta := dttype.ValueMetadata{Timestamp: now}
	metaJSON, _ := json.Marshal(meta)

	// deal expect
	if err := isTwinValueDiff(twin, msgTwin, false); err != nil {
		return nil, err
	}
	expectedValue := msgTwin.Expected.Value
	if twin.ExpectedVersion == nil {
		twin.ExpectedVersion = &dttype.TwinVersion{}
	}
	expectedVersion := twin.ExpectedVersion
	msgTwinExpectedVersion := msgTwin.ExpectedVersion
	if err := dealCompareTwinVersion(expectedVersion, msgTwinExpectedVersion, msgTwin.Metadata.Type); err != nil {
		return nil, err
	}
	expectedVersionJSON, _ := json.Marshal(expectedVersion)
	cols["expected"] = expectedValue
	cols["expected_meta"] = string(metaJSON)
	cols["expected_version"] = string(expectedVersionJSON)
	if twin.Expected == nil {
		twin.Expected = &dttype.TwinValue{}
	}
	twin.Expected.Value = expectedValue
	twin.Expected.Metadata = &meta
	twin.ExpectedVersion = expectedVersion

	// deal actual
	if err := isTwinValueDiff(twin, msgTwin, true); err != nil {
		return nil, err
	}
	actualValue := msgTwin.Actual.Value
	meta = dttype.ValueMetadata{Timestamp: now}
	if twin.ActualVersion == nil {
		twin.ActualVersion = &dttype.TwinVersion{}
	}
	actualVersion := twin.ActualVersion
	msgTwinActualVersion := msgTwin.ActualVersion
	if err := dealCompareTwinVersion(actualVersion, msgTwinActualVersion, msgTwin.Metadata.Type); err != nil {
		return nil, err
	}
	actualVersionJSON, _ := json.Marshal(actualVersion)
	cols["actual"] = actualValue
	cols["actual_meta"] = string(metaJSON)
	cols["actual_version"] = string(actualVersionJSON)
	if twin.Actual == nil {
		twin.Actual = &dttype.TwinValue{}
	}
	twin.Actual.Value = actualValue
	twin.Actual.Metadata = &meta
	twin.ActualVersion = actualVersion

	if msgTwin.Optional != nil &&
		*msgTwin.Optional != *twin.Optional &&
		*twin.Optional {
		optional := *msgTwin.Optional
		cols["optional"] = optional
		twin.Optional = &optional
	}
	if msgTwin.Metadata != nil {
		msgMetaJSON, _ := json.Marshal(msgTwin.Metadata)
		twinMetaJSON, _ := json.Marshal(twin.Metadata)
		if string(msgMetaJSON) != string(twinMetaJSON) {
			meta := dttype.CopyMsgTwin(msgTwin, true)
			meta.Metadata.Type = ""
			metaJSON, _ := json.Marshal(meta.Metadata)
			cols["metadata"] = string(metaJSON)
			if twin.Metadata.Type == dtcommon.TypeDeleted {
				cols["attr_type"] = msgTwin.Metadata.Type
				twin.Metadata.Type = msgTwin.Metadata.Type
			}
		}
	} else {
		if twin.Metadata.Type == dtcommon.TypeDeleted {
			twin.Metadata = &dttype.TypeMetadata{Type: constants.DataTypeString}
			cols["attr_type"] = constants.DataTypeString
		}
	}

	return &dtclient.DeviceTwinUpdate{DeviceID: deviceID, Name: key, Cols: cols}, nil
}

func dealTwinUpdate(deviceID string, src, twins map[string]*dttype.MsgTwin) dttype.DealTwinResult {
	res := dttype.DealTwinResult{
		Add:    make([]dtclient.DeviceTwin, 0),
		Delete: make([]dtclient.DeviceDelete, 0),
		Update: make([]dtclient.DeviceTwinUpdate, 0),
		Result: make(map[string]*dttype.MsgTwin),
	}
	for key, twin := range twins {
		oldTwin, exists := src[key]
		if !exists {
			// src not found twin, and twin metadata type is delete, so continue
			if twin.Metadata != nil && twin.Metadata.Type == dtcommon.TypeDeleted {
				continue
			}
			// src not found twin, so add it
			if err := dealTwinAdd(&res.Add, deviceID, map[string]*dttype.MsgTwin{key: twin}); err != nil {
				return dttype.DealTwinResult{Err: err}
			}
			continue
		}
		// src found twin, but twin metadata type is delete, so remove it
		if twin.Metadata != nil && twin.Metadata.Type == dtcommon.TypeDeleted {
			res.Delete = append(res.Delete, dtclient.DeviceDelete{DeviceID: deviceID, Name: key})
			continue
		}
		// src found twin, metadata type is normal data type, so update it
		twinUpdate, err := dealTwinCompare(deviceID, key, oldTwin, twin)
		if err != nil {
			return dttype.DealTwinResult{Err: err}
		}
		if twinUpdate != nil {
			res.Update = append(res.Update, *twinUpdate)
		}
	}
	return res
}

func dealAttrUpdate(deviceID string, srcAttrs, msgAttrs map[string]*dttype.MsgAttr) dttype.DealAttrResult {
	if srcAttrs == nil {
		srcAttrs = make(map[string]*dttype.MsgAttr)
	}
	add := make([]dtclient.DeviceAttr, 0)
	deletes := make([]dtclient.DeviceDelete, 0)
	update := make([]dtclient.DeviceAttrUpdate, 0)
	result := make(map[string]*dttype.MsgAttr)

	for key, msgAttr := range msgAttrs {
		if attr, exist := srcAttrs[key]; exist {
			if msgAttr == nil {
				if *attr.Optional {
					deletes = append(deletes, dtclient.DeviceDelete{DeviceID: deviceID, Name: key})
					result[key] = nil
					delete(srcAttrs, key)
				}
				continue
			}
			isChange := false
			cols := make(map[string]interface{})
			result[key] = &dttype.MsgAttr{}
			if strings.Compare(attr.Value, msgAttr.Value) != 0 {
				attr.Value = msgAttr.Value

				cols["value"] = msgAttr.Value
				result[key].Value = msgAttr.Value

				isChange = true
			}
			if msgAttr.Metadata != nil {
				msgMetaJSON, _ := json.Marshal(msgAttr.Metadata)
				attrMetaJSON, _ := json.Marshal(attr.Metadata)
				if strings.Compare(string(msgMetaJSON), string(attrMetaJSON)) != 0 {
					cols["attr_type"] = msgAttr.Metadata.Type
					meta := dttype.CopyMsgAttr(msgAttr)
					attr.Metadata = meta.Metadata
					msgAttr.Metadata.Type = ""
					metaJSON, _ := json.Marshal(msgAttr.Metadata)
					cols["metadata"] = string(metaJSON)
					msgAttr.Metadata.Type = cols["attr_type"].(string)
					result[key].Metadata = meta.Metadata
					isChange = true
				}
			}
			if msgAttr.Optional != nil {
				if *msgAttr.Optional != *attr.Optional && *attr.Optional {
					optional := *msgAttr.Optional
					cols["optional"] = optional
					attr.Optional = &optional
					result[key].Optional = &optional
					isChange = true
				}
			}
			if isChange {
				update = append(update, dtclient.DeviceAttrUpdate{DeviceID: deviceID, Name: key, Cols: cols})
			} else {
				delete(result, key)
			}
		} else {
			deviceAttr := dttype.MsgAttrToDeviceAttr(key, msgAttr)
			deviceAttr.DeviceID = deviceID
			deviceAttr.Value = msgAttr.Value
			if msgAttr.Optional != nil {
				optional := *msgAttr.Optional
				deviceAttr.Optional = optional
			}
			if msgAttr.Metadata != nil {
				deviceAttr.AttrType = msgAttr.Metadata.Type
				msgAttr.Metadata.Type = ""
				metaJSON, _ := json.Marshal(msgAttr.Metadata)
				msgAttr.Metadata.Type = deviceAttr.AttrType
				deviceAttr.Metadata = string(metaJSON)
			}
			add = append(add, deviceAttr)
			srcAttrs[key] = msgAttr
			result[key] = msgAttr
		}

	}
	return dttype.DealAttrResult{Add: add, Delete: deletes, Update: update, Result: result, Err: nil}
}
