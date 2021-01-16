package internal

// docker.go implements the AppRunner interface to allow for
// apps to use Docker containers as the run platform
// This serves as a wrapper around the docker API so that this
// app server's API can abstract most of the details of the container
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

// Create and start the docker container as well as set up
// jobs to manage the container. The container will be stopped
// after 1 minute of inactivity and will be removed after 15 minutes.
// It will also check every second whether the container is running
// by sending a request to port 9003.
func (d *DockerContainerRunner) Create() error {
	if err := d.create(); err != nil {
		return err
	}

	if err := d.start(); err != nil {
		return err
	}

	// health check
	startJob, err := d.jobs.AddFunc("@every 1s", d.checkIsRunning)
	if err != nil {
		return err
	}

	// Stop after inactivity
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

	// Evict after long period of inactivity
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

// Cleanup will remove everything related to the specified container.
// The container will be stopped and removed, and any jobs related
// to the container will also be removed from the job queue
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

	d.jobs.Stop()
	d.jobHandles = make(map[string]cron.EntryID)

	return nil
}

func (d *DockerContainerRunner) IsReady() bool {
	return d.IsRunning
}

func (d *DockerContainerRunner) BlockUntilReady() {
	if !d.IsRunning {
		<-d.ready
	}
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
