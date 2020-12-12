package internal

import (
	"net/http/httputil"
	"net/url"
	"time"
)

type containerInstance struct {
	Id string `json:"Id"`

	LastInvocation time.Time `json:"LastInvocation"`
	IsRunning      bool      `json:"IsRunning"`

	DockerID    string            `json:"DockerId"`
	Volume      string            `json:"Volume"`
	Environment map[string]string `json:"Environment"`

	Port        int     `json:"Port"`
	FrontendUrl url.URL `json:"FrontendUrl"`
	BackendUrl  url.URL `json:"BackendUrl"`
	NginxConf   string  `json:"NginxConf"`

	proxy *httputil.ReverseProxy
}
