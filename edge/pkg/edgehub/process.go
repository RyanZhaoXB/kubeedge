package edgehub

import (
	"fmt"
	"time"

	"github.com/kubeedge/kubeedge/common/constants"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/msghandler"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

var groupMap = map[string]string{
	"resource": modules.MetaGroup,
	"twin":     modules.TwinGroup,
	"func":     modules.MetaGroup,
	"user":     modules.BusGroup,
}

func (eh *EdgeHub) initial() (err error) {
	cloudHubClient, err := clients.GetClient()
	if err != nil {
		return err
	}

	eh.chClient = cloudHubClient

	return nil
}

func isSyncResponse(msgID string) bool {
	return msgID != ""
}

func init() {
	handler := &defaultHandler{}
	msghandler.RegisterHandler(handler)
}

type defaultHandler struct {
}

func (*defaultHandler) Filter(message *model.Message) bool {
	group := message.GetGroup()
	return group == messagepkg.ResourceGroupName || group == constants.ResourceGroupDeviceManager ||
		group == messagepkg.FuncGroupName || group == constants.UserGroup
}

func (*defaultHandler) Process(message *model.Message, clientHub clients.Adapter) error {
	srcGroup := message.GetGroup()
	group := ""
	switch srcGroup {
	case messagepkg.ResourceGroupName:
		group = modules.MetaGroup
	case constants.ResourceGroupDeviceManager:
		group = constants.ResourceGroupDeviceManager
		// TODO send message to mapper
	case messagepkg.FuncGroupName:
		group = modules.MetaGroup
	case constants.UserGroup:
		group = modules.BusGroup
	}

	if isSyncResponse(message.GetParentID()) {
		beehiveContext.SendResp(*message)
		return nil
	}
	if srcGroup == constants.UserGroup && message.GetSource() == constants.RouterSourceEventBus {
		beehiveContext.Send(modules.EventBusModuleName, *message)
	} else if srcGroup == constants.UserGroup && message.GetSource() == constants.RouterSourceServiceBus {
		beehiveContext.Send(modules.ServiceBusModuleName, *message)
	} else {
		beehiveContext.SendToGroup(group, *message)
	}
	return nil
}

func (eh *EdgeHub) dispatch(message model.Message) error {
	// handler for msg.
	return msghandler.ProcessHandler(message, eh.chClient)
}

func (eh *EdgeHub) routeToEdge() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub RouteToEdge stop")
			return
		default:
		}
		message, err := eh.chClient.Receive()
		if err != nil {
			klog.Errorf("websocket read error: %v", err)
			eh.reconnectChan <- struct{}{}
			return
		}

		klog.V(4).Infof("[edgehub/routeToEdge] receive msg from cloud, msg:% +v", message)
		err = eh.dispatch(message)
		if err != nil {
			klog.Errorf("failed to dispatch message, discard: %v", err)
		}
	}
}

func (eh *EdgeHub) sendToCloud(message model.Message) error {
	eh.keeperLock.Lock()
	klog.V(4).Infof("[edgehub/sendToCloud] send msg to cloud, msg: %+v", message)
	err := eh.chClient.Send(message)
	eh.keeperLock.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send message, error: %v", err)
	}

	return nil
}

func (eh *EdgeHub) routeToCloud() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub RouteToCloud stop")
			return
		default:
		}
		message, err := beehiveContext.Receive(modules.EdgeHubModuleName)
		if err != nil {
			klog.Errorf("failed to receive message from edge: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// post message to cloud hub
		err = eh.sendToCloud(message)
		if err != nil {
			klog.Errorf("failed to send message to cloud: %v", err)
			eh.reconnectChan <- struct{}{}
			return
		}
	}
}

func (eh *EdgeHub) keepalive() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub KeepAlive stop")
			return
		default:
		}
		msg := model.NewMessage("").
			BuildRouter(modules.EdgeHubModuleName, "resource", "node", messagepkg.OperationKeepalive).
			FillBody("ping")

		// post message to cloud hub
		err := eh.sendToCloud(*msg)
		if err != nil {
			klog.Errorf("websocket write error: %v", err)
			eh.reconnectChan <- struct{}{}
			return
		}

		time.Sleep(time.Duration(config.Config.Heartbeat) * time.Second)
	}
}

func (eh *EdgeHub) pubConnectInfo(isConnected bool) {
	// var info model.Message
	content := connect.CloudConnected
	if !isConnected {
		content = connect.CloudDisconnected
	}

	for _, group := range groupMap {
		message := model.NewMessage("").BuildRouter(messagepkg.SourceNodeConnection, group,
			messagepkg.ResourceTypeNodeConnection, messagepkg.OperationNodeConnection).FillBody(content)
		beehiveContext.SendToGroup(group, *message)
	}
}

func (eh *EdgeHub) ifRotationDone() {
	if eh.certManager.RotateCertificates {
		for {
			<-eh.certManager.Done
			eh.reconnectChan <- struct{}{}
		}
	}
}
