package internal

import "github.com/docker/docker/client"

type Global struct {
	ContainerMgr *ContainerManager
	Logger       *Logger
	Docker       *client.Client
}

var G *Global
