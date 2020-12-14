package internal

import (
	"net/http/httputil"
	"net/url"
	"time"
)

type ContainerManager struct {
	containers map[string]*containerInstance
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
		return nil, &mgrError{nil, "Container not found", mgrErrorExists}
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
			DockerName:  n.DockerName,
			Dir:         n.Dir,
			Environment: n.Environment,
			FrontendUrl: *frontendUrl,
			BackendUrl:  *backendUrl,
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
			Id:          id,
			Dir:         n.Dir,
			Environment: n.Environment,
		}
	}

	return mgr.containers[id]
}

func (mgr *ContainerManager) exists(id string) bool {
	_, ok := mgr.containers[id]
	return ok
}

func (mgr *ContainerManager) delete(id string) error {
	if _, ok := mgr.containers[id]; ok {
		delete(mgr.containers, id)
	} else {
		return &mgrError{nil, "Container not found", mgrErrorNotFound}
	}

	return nil
}

func (mgr *ContainerManager) StopContainers(limit time.Duration) error {
	cutoff := time.Now().Add(-limit)
	errors := &mgrError{}

	for _, c := range mgr.containers {
		if c.IsRunning && c.LastInvocation.Before(cutoff) {
			err := stopContainer(c)
			if err != nil {
				errors.errs = append(errors.errs, err)
			}
			c.IsRunning = false
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
			}
			c.DockerID = ""
		}
	}

	return nil
}
