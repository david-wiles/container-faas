package internal

import (
	"net/http/httputil"
	"net/url"
	"time"
)

type containerInstance struct {
	Id             string    `json:"Id"`             // Unique ID for this container instance. Matches the id used in ContainerMgr
	LastInvocation time.Time `json:"LastInvocation"` // Time of the last container invocation
	IsRunning      bool      `json:"IsRunning"`      // Indicates whether a docker container is currently running

	Image       string   `json:"Image"`       // Name of the image to use when creating this container
	DockerID    string   `json:"DockerId"`    // Docker ID, obtained once a container has been created
	DockerName  string   `json:"DockerName"`  // Unique name of the container. Should match the Id in most cases
	Dir         string   `json:"Dir"`         // Directory of the app files on the server
	Environment []string `json:"Environment"` // Any environment variables

	FrontendUrl url.URL `json:"FrontendUrl"` // User-facing or proxy-facing url
	BackendUrl  url.URL `json:"BackendUrl"`  // The URL of the container, used within the docker network

	Port      int    `json:"Port"`      // The port that nginx will use in the server block for this container
	NginxConf string `json:"NginxConf"` // The file containing the .conf file for this container

	proxy *httputil.ReverseProxy // Instance of a reverse proxy facing the docker container
}
