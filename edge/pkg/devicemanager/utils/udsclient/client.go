package udsclient

import (
"context"
"crypto/tls"
"io"
"net"
"net/http"
"time"

"k8s.io/klog/v2"
)

const (
	defaultConnectTimeout            = 30 * time.Second
	defaultKeepAliveTimeout          = 30 * time.Second
	defaultResponseReadTimeout       = 300 * time.Second
	defaultMaxIdleConnectionsPerHost = 3
	SockPath                         = "/root/data/test.sock"
)

var (
	connectTimeout            = defaultConnectTimeout
	keepaliveTimeout          = defaultKeepAliveTimeout
	responseReadTimeout       = defaultResponseReadTimeout
	maxIdleConnectionsPerHost = defaultMaxIdleConnectionsPerHost
)

// NewHTTPClient create new client
func NewHTTPClient(sockPath string) *http.Client {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   connectTimeout,
				KeepAlive: keepaliveTimeout,
			}).Dial("unix", sockPath)
		},
		MaxIdleConnsPerHost:   maxIdleConnectionsPerHost,
		ResponseHeaderTimeout: responseReadTimeout,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	klog.Infof("tlsConfig InsecureSkipVerify true")
	return &http.Client{Transport: transport}
}

// SendRequest sends a http request and return the resp info
func SendRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// BuildRequest Creates a HTTP request.
func BuildRequest(method string, urlStr string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
