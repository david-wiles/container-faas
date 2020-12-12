package internal

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type dockerErrorType int

const (
	dockerErrorNoType dockerErrorType = 0
	dockerErrorCreate dockerErrorType = 1
	dockerErrorStart  dockerErrorType = 2
	dockerErrorStop   dockerErrorType = 3
	dockerErrorRemove dockerErrorType = 4
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
	dockerResp, err := G.Docker.ContainerCreate(ctx, &container.Config{}, nil, nil, nil, "")
	if err != nil {
		return &dockerError{err, "Could not create docker container", dockerErrorCreate}
	}

	c.DockerID = dockerResp.ID
	return nil
}

func startContainer(c *containerInstance) error {
	ctx := context.Background()

	if err := G.Docker.ContainerStart(ctx, c.DockerID, types.ContainerStartOptions{}); err != nil {
		return &dockerError{err, "Could not start docker container", dockerErrorStart}
	}

	return nil
}

func stopContainer(c *containerInstance) error {
	ctx := context.Background()

	if err := G.Docker.ContainerStop(ctx, c.DockerID, nil); err != nil {
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
