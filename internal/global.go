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

	NginxAppDir string
	UseNginx    bool
}

// Parse all arguments. Passed arguments take precedence over environment variables
func ParseArgs(addr, stopTimeout, startTimeout, network string, useNginx bool, logLevel int) (*Global, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	if addr == "" {
		addr = os.Getenv("ADDR")
	}

	if stopTimeout == "" {
		stopTimeout = os.Getenv("DOCKER_STOP_TIMEOUT")
	}

	if startTimeout == "" {
		startTimeout = os.Getenv("CONTAINER_START_TIMEOUT")
	}

	if network == "" {
		network = os.Getenv("DOCKER_NETWORK")
	}

	dockerStopTimeout, err := time.ParseDuration(stopTimeout)
	if err != nil {
		return nil, err
	}

	containerStartTimeout, err := time.ParseDuration(startTimeout)
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
			level:    LogLevel(logLevel),
		},
		Docker:                docker,
		Addr:                  addr,
		DockerStopTimeout:     dockerStopTimeout,
		ContainerStartTimeout: containerStartTimeout,
		DockerNetwork:         network,
		NginxAppDir:           "/etc/nginx/apps/",
		UseNginx:              useNginx,
	}, nil
}

var G *Global
