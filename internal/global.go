package internal

import (
	"github.com/docker/docker/client"
	"os"
	"time"
)

type Global struct {
	ContainerMgr *ContainerManager
	Logger       *Logger
	Docker       *client.Client

	Addr                  string
	DockerStopTimeout     time.Duration
	ContainerStartTimeout time.Duration
	DockerNetwork         string
}

func ParseArgs() (*Global, error) {
	// TODO determine correct opts
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	dockerStopTimeout, err := time.ParseDuration(os.Getenv("DOCKER_STOP_TIMEOUT"))
	containerStartTimeout, err := time.ParseDuration(os.Getenv("CONTAINER_START_TIMEOUT"))
	if err != nil {
		return nil, err
	}

	return &Global{
		ContainerMgr:          &ContainerManager{},
		Logger:                &Logger{},
		Docker:                docker,
		Addr:                  os.Getenv("ADDR"),
		DockerStopTimeout:     dockerStopTimeout,
		ContainerStartTimeout: containerStartTimeout,
		DockerNetwork:         os.Getenv("DOCKER_NETWORK"),
	}, nil
}

var G *Global
