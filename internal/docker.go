package internal

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

type dockerErrorType int

const (
	dockerErrorNoType          dockerErrorType = 0
	dockerErrorContainerCreate dockerErrorType = 1
	dockerErrorStart           dockerErrorType = 2
	dockerErrorStop            dockerErrorType = 3
	dockerErrorRemove          dockerErrorType = 4
	dockerErrorNetworkCreate   dockerErrorType = 5
	dockerErrorImageBuild      dockerErrorType = 6
	dockerErrorContainerList   dockerErrorType = 7
)

type dockerError struct {
	oErr error
	msg  string
	t    dockerErrorType
}

func (err *dockerError) Error() string {
	if err != nil {
		return err.msg + " " + err.oErr.Error()
	} else {
		return ""
	}
}

func runContainer(c *containerInstance) error {
	err := createContainer(c)
	if err != nil {
		return err
	}

	err = startContainer(c)
	if err != nil {
		return err
	}

	return nil
}

func createContainer(c *containerInstance) error {
	ctx := context.Background()
	dockerResp, err := G.Docker.ContainerCreate(ctx,
		&container.Config{
			Env:   c.Environment,
			Image: c.Image,
		}, &container.HostConfig{
			Binds: []string{
				c.Dir + ":/home/app",
			},
		}, nil, nil, c.DockerName)
	if err != nil {
		return &dockerError{err, "Could not create docker container", dockerErrorContainerCreate}
	}

	c.DockerID = dockerResp.ID

	return nil
}

func startContainer(c *containerInstance) error {
	ctx := context.Background()

	if err := G.Docker.ContainerStart(ctx, c.DockerID, types.ContainerStartOptions{}); err != nil {
		return &dockerError{err, "Could not start docker container", dockerErrorStart}
	}

	if err := G.Docker.NetworkConnect(ctx, G.DockerNetwork, c.DockerID, &network.EndpointSettings{}); err != nil {
		_ = G.Docker.ContainerStop(ctx, c.DockerID, nil)
		return &dockerError{err, "Could not connect container to network", dockerErrorStart}
	}

	return nil
}

func stopContainer(c *containerInstance) error {
	ctx := context.Background()

	if err := G.Docker.ContainerStop(ctx, c.DockerID, &G.DockerStopTimeout); err != nil {
		return &dockerError{err, "Could not stop docker container", dockerErrorStop}
	}

	return nil
}

func removeContainer(c *containerInstance) error {
	ctx := context.Background()

	if err := G.Docker.ContainerRemove(ctx, c.DockerID, types.ContainerRemoveOptions{}); err != nil {
		return &dockerError{err, "Could not remove container", dockerErrorRemove}
	}

	return nil
}

func getContainerName(id string) ([]string, error) {
	ctx := context.Background()

	containers, err := G.Docker.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("id", id)),
	})
	if err != nil {
		return nil, err
	}

	if len(containers) > 1 {
		return nil, &dockerError{nil, "Id passed is not unique", dockerErrorContainerList}
	}

	return containers[0].Names, nil
}
