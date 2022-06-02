package httpclient

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

const (
	defaultConnectTimeout            = 30 * time.Second
	defaultKeepAliveTimeout          = 30 * time.Second
	defaultResponseReadTimeout       = 300 * time.Second
	defaultMaxIdleConnectionsPerHost = 3
)

var (
	connectTimeout            = defaultConnectTimeout
	keepaliveTimeout          = defaultKeepAliveTimeout
	responseReadTimeout       = defaultResponseReadTimeout
	maxIdleConnectionsPerHost = defaultMaxIdleConnectionsPerHost
)

// NewHTTPClient create new client
func NewHTTPClient() *http.Client {
	transport := &http.Transport{
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

func HttpRequest(url, method string, body []byte) ([]byte, error) {
	client := NewHTTPClient()
	req, err := BuildRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		klog.Errorf("fail to build request with error : %+v", err)
		return nil, err
	}
	res, err := SendRequest(req, client)
	if err != nil {
		klog.Errorf("fail to send request with error : %+v", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		klog.Errorf("fail to get resp with code : %d", res.StatusCode)
		return nil, fmt.Errorf("fail to get mapper info")
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		klog.Errorf("fail to read data with error : %+v", err)
		return nil, err
	}

	return data, nil
}
