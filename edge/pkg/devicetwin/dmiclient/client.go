package dmiclient

import (
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"

	pb "github.com/kubeedge/kubeedge/edge/pkg/apis/dmi/v1"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

const (
	unix string = "unix"
)

func GenerateDMIClient(sockPath string) (pb.DeviceMapperServiceClient, context.Context, *grpc.ClientConn, context.CancelFunc, error) {
	dialer := func(addr string, t time.Duration) (net.Conn, error) {
		return net.Dial(unix, addr)
	}

	conn, err := grpc.Dial(sockPath, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		klog.Errorf("did not connect: %v\n", err)
		return nil, nil, nil, nil, err
	}

	c := pb.NewMapperClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	return c, ctx, conn, cancel, nil
}

func CreateDeviceRequest(device *v1alpha2.Device, model *v1alpha2.DeviceModel) (*pb.CreateDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}
	return &pb.CreateDeviceRequest{
		Config: &pb.DeviceConfig{
			Model:  m,
			Device: d,
		},
	}, nil
}

func RemoveDeviceRequest(deviceName string) (*pb.RemoveDeviceRequest, error) {
	return &pb.RemoveDeviceRequest{
		DeviceName: deviceName,
	}, nil
}

func UpdateDeviceRequest(device *v1alpha2.Device, model *v1alpha2.DeviceModel) (*pb.UpdateDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateDeviceRequest{
		Config: &pb.DeviceConfig{
			Model:  m,
			Device: d,
		},
	}, nil
}
