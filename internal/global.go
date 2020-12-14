package internal

import (
	"github.com/docker/docker/client"
	"os"
	"strconv"
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

	UseNginx bool
}

func ParseArgs() (*Global, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	dockerStopTimeout, err := time.ParseDuration(os.Getenv("DOCKER_STOP_TIMEOUT"))
	if err != nil {
		return nil, err
	}
	containerStartTimeout, err := time.ParseDuration(os.Getenv("CONTAINER_START_TIMEOUT"))
	if err != nil {
		return nil, err
	}
	level, err := strconv.Atoi(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return nil, err
	}

	return &Global{
		ContainerMgr: &ContainerManager{
			containers: make(map[string]*containerInstance),
		},
		Logger: &Logger{
			infoLog:  os.Stdout,
			errorLog: os.Stderr,
			level:    LogLevel(level),
		},
		Docker:                docker,
		Addr:                  os.Getenv("ADDR"),
		DockerStopTimeout:     dockerStopTimeout,
		ContainerStartTimeout: containerStartTimeout,
		DockerNetwork:         os.Getenv("DOCKER_NETWORK"),
		UseNginx:              os.Getenv("USE_NGINX") == "1",
	}, nil
}

var G *Global
