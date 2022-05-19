package udsserver

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	// DefaultBufferSize represents default buffer size
	DefaultBufferSize = 10480
)

// UnixDomainSocket struct
type UnixDomainSocket struct {
	filename   string
	buffersize int
	handler    func(string) string
}

// NewUnixDomainSocket create new socket
func NewUnixDomainSocket(filename string, buffersize ...int) *UnixDomainSocket {
	size := DefaultBufferSize
	if len(buffersize) != 0 {
		size = buffersize[0]
	}
	return &UnixDomainSocket{filename: filename, buffersize: size}
}

// parseEndpoint parses endpoint
func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

// SetContextHandler set handler for server
func (us *UnixDomainSocket) SetContextHandler(f func(string) string) {
	us.handler = f
}

// StartServer start for server
func (us *UnixDomainSocket) StartServer(container *restful.Container) error {
	proto, addr, err := parseEndpoint(us.filename)
	if err != nil {
		klog.Errorf("failed to parseEndpoint: %v", err)
		return err
	}
	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) { //nolint: vetshadow
			klog.Errorf("failed to remove addr: %v", err)
			return err
		}
	}

	// Listen
	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Errorf("failed to listen addr: %v", err)
		return err
	}
	defer listener.Close()
	klog.Infof("listening on: %v", listener.Addr())

	server := http.Server{
		Handler: container,
	}
	return server.Serve(listener)
}
