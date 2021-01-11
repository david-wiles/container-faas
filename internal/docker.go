package internal

import (
	"context"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/robfig/cron/v3"
	"net/http"
	"net/http/httputil"
	"time"
)

type DockerContainerRunner struct {
	Image      string   `json:"Image"`      // Name of the image to use when creating this container
	Cmd        []string `json:"Cmd"`        // Command to execute when starting the container
	DockerID   string   `json:"DockerId"`   // Docker ID, obtained once a container has been created
	DockerName string   `json:"DockerName"` // Unique name of the container. Should match the Id in most cases
	Dir        string   `json:"Dir"`        // Directory of the app files on the server
	Env        []string `json:"Env"`        // Any environment variables
	IsRunning  bool     `json:"isRunning"`  // Indicates whether this docker container is running

	jobHandles []cron.EntryID
	proxy      *httputil.ReverseProxy // Reverse proxy used to route requests to the app
}

func NewDockerContainer(image, dockerId, dockerName, dir string, cmd, env []string) *DockerContainerRunner {
	return &DockerContainerRunner{
		Image:      image,
		Cmd:        cmd,
		DockerID:   dockerId,
		DockerName: dockerName,
		Dir:        dir,
		Env:        env,
	}
}

func (d *DockerContainerRunner) Create() error {
	return d.start()
}

func (d *DockerContainerRunner) Cleanup() error {
	if d.IsRunning {
		err := d.stop()
		if err != nil {
			return err
		}
	}

	err := d.remove()
	if err != nil {
		return err
	}

	for _, job := range d.jobHandles {
		G.Jobs.Remove(job)
	}

	return nil
}

// Set up jobs to automatically stop or remove container when there have been no recent invocations
func (d *DockerContainerRunner) InitCleanupJobs(lastInvocation *time.Time) error {
	stopJob, err := G.Jobs.AddFunc("@every 1m", func() {
		cutoff := time.Now().Add(-time.Minute * 15)
		if lastInvocation.Before(cutoff) {
			err := d.stop()
			if err != nil {
				G.Logger.LogError(err)
			}
			d.IsRunning = false
		}
	})
	if err != nil {
		return err
	}

	removeJob, err := G.Jobs.AddFunc("@every 15m", func() {
		cutoff := time.Now().Add(-time.Hour)
		if lastInvocation.Before(cutoff) {
			err := d.remove()
			if err != nil {
				G.Logger.LogError(err)
				return
			}
			d.DockerID = ""
		}
	})
	if err != nil {
		return err
	}

	d.jobHandles = []cron.EntryID{stopJob, removeJob}
	return nil
}

func (d *DockerContainerRunner) SetIsReady() {
	d.IsRunning = true
}

func (d *DockerContainerRunner) IsReady() bool {
	return d.IsRunning
}

func (d *DockerContainerRunner) Invoke(w http.ResponseWriter, r *http.Request) {
	d.proxy.ServeHTTP(w, r)
}

func (d *DockerContainerRunner) run() error {
	err := d.create()
	if err != nil {
		return err
	}

	err = d.start()
	if err != nil {
		return err
	}

	return nil
}

func (d *DockerContainerRunner) create() error {
	ctx := context.Background()
	dockerResp, err := G.Docker.ContainerCreate(ctx,
		&container.Config{
			Env:        d.Env,
			Image:      d.Image,
			Cmd:        d.Cmd,
			Entrypoint: []string{"docker-entrypoint.sh"},
		}, &container.HostConfig{
			Binds: []string{
				d.Dir + ":/home/app",
			},
		}, nil, nil, d.DockerName)
	if err != nil {
		return errors.New("Could not create docker container")
	}

	if err := G.Docker.NetworkConnect(ctx, G.DockerNetwork, dockerResp.ID, &network.EndpointSettings{}); err != nil {
		_ = G.Docker.ContainerStop(ctx, dockerResp.ID, nil)
		return errors.New("Could not connect container to network")
	}

	d.DockerID = dockerResp.ID

	return nil
}

func (d *DockerContainerRunner) start() error {
	ctx := context.Background()

	if err := G.Docker.ContainerStart(ctx, d.DockerID, types.ContainerStartOptions{}); err != nil {
		return errors.New("Could not start docker container")
	}

	return nil
}

func (d *DockerContainerRunner) stop() error {
	ctx := context.Background()

	if err := G.Docker.ContainerStop(ctx, d.DockerID, &G.StopTimeout); err != nil {
		return errors.New("Could not stop docker container")
	}

	return nil
}

func (d *DockerContainerRunner) remove() error {
	ctx := context.Background()

	if err := G.Docker.ContainerRemove(ctx, d.DockerID, types.ContainerRemoveOptions{}); err != nil {
		return errors.New("Could not remove container")
	}

	return nil
}
