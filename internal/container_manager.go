package internal

import (
	"math/rand"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

type ContainerManager struct {
	containers map[string]*containerInstance
	ports      [64511]bool
}

type mgrErrorType int

const (
	mgrErrorNoType   mgrErrorType = 0
	mgrErrorExists   mgrErrorType = 1
	mgrErrorNotFound mgrErrorType = 2
)

type mgrError struct {
	errs []error
	msg  string
	t    mgrErrorType
}

func (err *mgrError) Error() string {
	if err != nil {
		return err.msg
	} else {
		return ""
	}
}

func ContainerNotFound(err error) bool {
	if mgrErr, ok := err.(*mgrError); ok {
		return mgrErr.t == mgrErrorNotFound
	}
	return false
}

func (mgr *ContainerManager) get(id string) (*containerInstance, error) {
	if c, ok := mgr.containers[id]; ok {
		return c, nil
	} else {
		return nil, &mgrError{nil, "Container not found", mgrErrorNotFound}
	}
}

func (mgr *ContainerManager) create(id string, n containerInstance) (*containerInstance, error) {
	if _, ok := mgr.containers[id]; !ok {
		backendUrl, err := url.Parse("http://" + n.DockerName + ":8080")
		if err != nil {
			return nil, err
		}

		frontendUrl, err := url.Parse("http://" + G.Addr + "/container/" + id)
		if err != nil {
			return nil, err
		}

		c := &containerInstance{
			Id:          id,
			Image:       n.Image,
			Cmd:         n.Cmd,
			DockerName:  n.DockerName,
			Dir:         n.Dir,
			Env:         n.Env,
			FrontendUrl: *frontendUrl,
			BackendUrl:  *backendUrl,
			Port:        mgr.reservePort(),
			NginxConf:   G.NginxAppDir + n.DockerName + ".conf",
		}

		c.proxy = httputil.NewSingleHostReverseProxy(&c.BackendUrl)

		mgr.containers[id] = c
	} else {
		return nil, &mgrError{nil, "Container already exists", mgrErrorExists}
	}

	return mgr.containers[id], nil
}

func (mgr *ContainerManager) update(id string, n containerInstance) (*containerInstance, error) {
	if c, ok := mgr.containers[id]; ok {
		// Update non-replaceable fields from old container and replace entry
		n.Id = c.Id
		mgr.containers[id] = &n
	} else {
		return nil, &mgrError{nil, "Container not found", mgrErrorNotFound}
	}

	return mgr.containers[id], nil
}

func (mgr *ContainerManager) updateOrCreate(id string, n containerInstance) *containerInstance {
	if _, ok := mgr.containers[id]; ok {
		_, _ = mgr.update(id, n)
	} else {
		mgr.containers[id] = &containerInstance{
			Id:  id,
			Dir: n.Dir,
			Env: n.Env,
		}
	}

	return mgr.containers[id]
}

func (mgr *ContainerManager) exists(id string) bool {
	_, ok := mgr.containers[id]
	return ok
}

func (mgr *ContainerManager) delete(id string) error {
	if c, ok := mgr.containers[id]; ok {
		err := os.Remove(c.NginxConf)
		if err != nil {
			return err
		}
		err = nginxReload()
		if err != nil {
			return err
		}
		mgr.ports[c.Port] = false
		delete(mgr.containers, id)
	} else {
		return &mgrError{nil, "Container not found", mgrErrorNotFound}
	}

	return nil
}

// Reset all docker-related attributes. This should be used in the case of an error to
// prevent the program from getting stuck in a bad state inconsistent with Docker
func (mgr *ContainerManager) reset(id string) error {
	if c, ok := mgr.containers[id]; ok {

		c.LastInvocation = time.Time{}
		c.IsRunning = false
		c.DockerID = ""
		c.DockerName = ""
		c.BackendUrl = url.URL{}

		return nil
	} else {
		return &mgrError{nil, "Container not found", mgrErrorNotFound}
	}
}

func (mgr *ContainerManager) StopContainers(limit time.Duration) error {
	cutoff := time.Now().Add(-limit)
	errors := &mgrError{}

	for _, c := range mgr.containers {
		if c.IsRunning && c.LastInvocation.Before(cutoff) {
			err := stopContainer(c)
			if err != nil {
				errors.errs = append(errors.errs, err)
			} else {
				c.IsRunning = false
			}
		}
	}

	return nil
}

func (mgr *ContainerManager) EvictContainers(limit time.Duration) error {
	cutoff := time.Now().Add(-limit)
	errors := &mgrError{}

	for _, c := range mgr.containers {
		if !c.IsRunning && c.LastInvocation.Before(cutoff) {
			err := removeContainer(c)
			if err != nil {
				errors.errs = append(errors.errs, err)
			} else {
				c.DockerID = ""
			}
		}
	}

	return nil
}

// TODO mutex
func (mgr *ContainerManager) reservePort() int {
	port := rand.Intn(6000-5000) + 5000

	for mgr.ports[port] {
		port = (((port - 5000) + 1) % 1000) + 5000
	}

	mgr.ports[port] = true
	return port
}
