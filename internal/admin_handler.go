package internal

import (
	"encoding/json"
	"net/http"
	"strings"
)

type AdminHandler struct{}

func (h AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
	case "POST":
		h.post(w, r)
	case "DELETE":
		h.delete(w, r)
	default:
		HTTPError(w, "HTTP Method not supported", 400)
	}
}

func (AdminHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := trimPath("/admin/", r)
	if err != nil {
		HTTPError(w, "Resource not found", 404)
		return
	}

	c, err := G.ContainerMgr.get(id)
	if err != nil {
		if ContainerNotFound(err) {
			G.Logger.Warning(err.Error())
			HTTPError(w, err.Error(), 404)
			return
		} else {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	b, err := json.Marshal(c)
	if err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}

type containerPostRequest struct {
	Image string   `json:"Image"`
	Cmd   string   `json:"Cmd"`
	Dir   string   `json:"Dir"`
	Env   []string `json:"Env"`
}

func (AdminHandler) post(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	reqBody := &containerPostRequest{}

	id, err := trimPath("/admin/", r)
	if err != nil {
		HTTPError(w, "Resource not found", 404)
		return
	}

	if G.ContainerMgr.exists(id) {
		HTTPError(w, "Container already exists", 200)
		return
	}

	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(reqBody)
	if err != nil {
		G.Logger.LogError(err)
		HTTPError(w, "Could not parse request body: "+err.Error(), 500)
		return
	}

	// Create container entry
	tmp := containerInstance{
		Image:      reqBody.Image,
		Cmd:        strings.Split(reqBody.Cmd, " "),
		DockerName: id,
		Dir:        reqBody.Dir,
		Env:        reqBody.Env,
	}
	c, err := G.ContainerMgr.create(id, tmp)
	if err != nil {
		_ = G.ContainerMgr.delete(id)
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	// Create docker container
	err = runContainer(c)
	if err != nil {

		_ = removeContainer(c)
		_ = G.ContainerMgr.reset(id)

		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	if G.UseNginx {
		err = writeNginxConf(c.NginxConf, c.Port, c.FrontendUrl.String())
		if err != nil {
			G.Logger.LogError(err)
			HTTPError(w, "Could not write Nginx configuration file: "+err.Error(), 500)
			return
		}

		err = nginxReload()
		if err != nil {
			G.Logger.LogError(err)
			HTTPError(w, "Could not reload nginx: "+err.Error(), 500)
			return
		}
	}

	G.Logger.Info("Successfully built container")

	b, err := json.Marshal(c)
	if err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}

func (AdminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := trimPath("/admin/", r)
	if err != nil {
		HTTPError(w, "Resource not found", 404)
		return
	}

	c, err := G.ContainerMgr.get(id)
	if err != nil {
		if ContainerNotFound(err) {
			G.Logger.Warning(err.Error())
			HTTPError(w, err.Error(), 404)
			return
		} else {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	defer G.ContainerMgr.delete(id)

	if c.IsRunning {
		if err = stopContainer(c); err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	if err = removeContainer(c); err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
}
