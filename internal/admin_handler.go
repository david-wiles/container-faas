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

	if app, ok := G.AppMgr.Get(id); ok {
		b, err := json.Marshal(app)
		if err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(b)
	} else {
		G.Logger.Warning("App not found: " + id)
		HTTPError(w, "App not found", 404)
	}
}

type containerPostRequest struct {
	Image string   `json:"image"`
	Cmd   string   `json:"cmd"`
	Dir   string   `json:"dir"`
	Env   []string `json:"env"`
}

func (AdminHandler) post(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	reqBody := &containerPostRequest{}

	id, err := trimPath("/admin/", r)
	if err != nil {
		HTTPError(w, "Resource not found", 404)
		return
	}

	if _, ok := G.AppMgr.Get(id); ok {
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

	if app, ok := G.AppMgr.Create(App{Id: id}); ok {
		app.Runner = NewDockerContainer(
			reqBody.Image,
			"",
			id,
			reqBody.Dir,
			strings.Split(reqBody.Cmd, " "),
			reqBody.Env,
		)

		if err := app.Init(); err != nil {
			_ = app.Runner.Cleanup()
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}

		if err := initAppIngress(app); err != nil {
			_ = G.Ingress.Remove(app)
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}

		G.Logger.Info("Successfully built container")

		b, err := json.Marshal(app)
		if err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(b)
	} else {
		G.AppMgr.Delete(id)
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}
}

func (AdminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := trimPath("/admin/", r)
	if err != nil {
		HTTPError(w, "Resource not found", 404)
		return
	}

	if app, ok := G.AppMgr.Get(id); ok {
		if err = app.Runner.Cleanup(); err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}

		if err = removeAppIngress(app); err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}

		G.AppMgr.Delete(id)

		w.WriteHeader(200)

	} else {
		G.Logger.Warning("Container not found")
		HTTPError(w, "Resource not found", 404)
		return
	}
}

func initAppIngress(app *App) error {
	if err := G.Ingress.Write(app); err != nil {
		return err
	}

	if err := G.Ingress.Reload(); err != nil {
		return err
	}

	return nil
}

func removeAppIngress(app *App) error {
	if err := G.Ingress.Remove(app); err != nil {
		return err
	}

	if err := G.Ingress.Reload(); err != nil {
		return err
	}

	return nil
}
