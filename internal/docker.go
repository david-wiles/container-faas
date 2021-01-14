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
	"net/url"
	"time"
)

type DockerContainerRunner struct {
	appID    string
	dockerID string

	Image      string   `json:"Image"`      // Name of the image to use when creating this container
	Cmd        []string `json:"Cmd"`        // Command to execute when starting the container
	DockerName string   `json:"DockerName"` // Unique name of the container. Should match the ID in most cases
	Dir        string   `json:"Dir"`        // Directory of the app files on the server
	Env        []string `json:"Env"`        // Any environment variables
	IsRunning  bool     `json:"isRunning"`  // Indicates whether this docker container is running

	jobs       *cron.Cron
	jobHandles map[string]cron.EntryID
	ready      chan bool

	backendURL string
	proxy      *httputil.ReverseProxy // Reverse proxy used to route requests to the app
}

func NewDockerContainer(appID, image, dockerName, dir string, cmd, env []string) *DockerContainerRunner {
	return &DockerContainerRunner{
		appID:      appID,
		Image:      image,
		Cmd:        cmd,
		DockerName: dockerName,
		Dir:        dir,
		Env:        env,
		backendURL: "http://" + dockerName + ":8080",
		jobs:       cron.New(),
		jobHandles: make(map[string]cron.EntryID),
		ready:      make(chan bool, 1),
	}
}

func (d *DockerContainerRunner) Create() error {
	if err := d.create(); err != nil {
		return err
	}

	if err := d.start(); err != nil {
		return err
	}

	startJob, err := d.jobs.AddFunc("@every 1s", d.checkIsRunning)
	if err != nil {
		return err
	}

	stopJob, err := d.jobs.AddFunc("@every 1m", func() {
		app, _ := G.AppMgr.Get(d.appID)
		cutoff := time.Now().Add(-time.Minute * 15)
		if app.LastInvocation.Before(cutoff) {
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

	removeJob, err := d.jobs.AddFunc("@every 15m", func() {
		app, _ := G.AppMgr.Get(d.appID)
		cutoff := time.Now().Add(-time.Hour)
		if app.LastInvocation.Before(cutoff) {
			err := d.remove()
			if err != nil {
				G.Logger.LogError(err)
				return
			}
			d.dockerID = ""
		}
	})
	if err != nil {
		return err
	}

	d.jobHandles = map[string]cron.EntryID{
		"stop":   stopJob,
		"remove": removeJob,
		"start":  startJob,
	}

	d.jobs.Start()

	u, err := url.Parse(d.backendURL)
	if err != nil {
		return err
	}

	d.proxy = httputil.NewSingleHostReverseProxy(u)

	return nil
}

func (d *DockerContainerRunner) Cleanup() error {
	d.jobs.Stop()
	d.jobHandles = make(map[string]cron.EntryID)

	if d.IsRunning {
		err := d.stop()
		if err != nil {
			return err
		}
		d.IsRunning = false
	}

	err := d.remove()
	if err != nil {
		return err
	}

	return nil
}

func (d *DockerContainerRunner) IsReady() bool {
	return d.IsRunning
}

func (d *DockerContainerRunner) BlockUntilReady() {
	// If d.ready has been closed, then this will return without blocking
	<-d.ready
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

	d.dockerID = dockerResp.ID

	return nil
}

func (d *DockerContainerRunner) start() error {
	ctx := context.Background()

	if err := G.Docker.ContainerStart(ctx, d.dockerID, types.ContainerStartOptions{}); err != nil {
		return errors.New("Could not start docker container")
	}

	if _, ok := d.jobHandles["start"]; !ok {
		handle, err := d.jobs.AddFunc("@every 1s", d.checkIsRunning)
		if err != nil {
			return err
		}
		d.jobHandles["start"] = handle
	}

	return nil
}

func (d *DockerContainerRunner) stop() error {
	ctx := context.Background()

	if err := G.Docker.ContainerStop(ctx, d.dockerID, &G.StopTimeout); err != nil {
		return errors.New("Could not stop docker container")
	}

	// Whenever a container is stopped, we may need to wait for it to run again
	d.ready = make(chan bool, 1)

	return nil
}

func (d *DockerContainerRunner) remove() error {
	ctx := context.Background()

	if err := G.Docker.ContainerRemove(ctx, d.dockerID, types.ContainerRemoveOptions{}); err != nil {
		return errors.New("Could not remove container")
	}

	return nil
}

// Wait for the container to run and set the runner's state to ready
func (d *DockerContainerRunner) checkIsRunning() {
	select {
	case <-d.ready:
		// If no requests are waiting on this app, this select statement
		// will set the state to running and remove the job
		d.IsRunning = true
		d.jobs.Remove(d.jobHandles["start"])
		delete(d.jobHandles, "start")
		return
	default:
	}

	// Send a request to the health endpoint in the app
	resp, err := http.Get("http://" + d.DockerName + ":9003")
	if err != nil {
		G.Logger.LogError(err)
	} else if resp.StatusCode == 200 {
		d.ready <- true
	}
}
