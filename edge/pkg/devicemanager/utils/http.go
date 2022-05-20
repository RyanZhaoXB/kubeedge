package utils

import (
	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"
)

// not return err for meaningless golint and codex
func WriteHttpResp(resp *restful.Response, status int, value interface{}) {
	err := resp.WriteHeaderAndEntity(status, value)
	if err != nil {
		klog.Errorf("failed to write resp code and body, err: %v", err)
	}
}
