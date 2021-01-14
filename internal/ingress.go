package internal

import (
	"errors"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"text/template"
)

type IngressServer interface {
	Write(*App) (string, error) // Writes the settings from the app runner into the global ingress configuration
	Remove(*App) error          // Remove the settings saved for this app
	Reload() error              // Activates the new ingress settings
}

type NoIngress struct{}

func (NoIngress) Write(a *App) (string, error) { return a.frontendURL, nil }
func (NoIngress) Remove(*App) error            { return nil }
func (NoIngress) Reload() error                { return nil }

// NginxPorts represents an Nginx reverse proxy that uses a different port for each app
type NginxPorts struct {
	NginxAppDir string
	confMu      *sync.Mutex
	ports       [100]bool
	apps        map[string]confPortEntry
}

type confPortEntry struct {
	port int
	file string
}

// Write a new nginx conf file for the app using the app runner specified
func (n *NginxPorts) Write(app *App) (string, error) {
	const nginxTemplate = "server {\n\tlisten {{ .Port }};\n  \n\tlocation / {\n\t\tproxy_pass {{ .Url }}/;\n\t}\n}"

	file := path.Join(n.NginxAppDir, app.ID+".conf")

	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return "", err
	}
	defer f.Close()

	frontend, err := url.Parse("http://" + G.Addr + "/app/" + app.ID)
	if err != nil {
		return "", err
	}

	port, ok := n.reservePort()
	if !ok {
		return "", errors.New("Out of ingress space")
	}

	t := template.Must(template.New("conf").Parse(nginxTemplate))
	err = t.Execute(f, struct {
		Port int
		Url  string
	}{port, frontend.String()})
	if err != nil {
		return "", err
	}

	n.confMu.Lock()
	n.apps[app.ID] = confPortEntry{port, file}
	n.confMu.Unlock()

	return ":" + strconv.Itoa(port), nil
}

// Remove any configuration related to the specified app
func (n *NginxPorts) Remove(app *App) error {
	n.confMu.Lock()
	defer n.confMu.Unlock()

	if entry, ok := n.apps[app.ID]; ok {
		if err := os.Remove(entry.file); err != nil {
			return err
		}
		n.ports[entry.port-5000] = false
		delete(n.apps, app.ID)
	}
	return nil
}

// Reload the current nginx instance by exec'ing a new process
func (*NginxPorts) Reload() error {
	cmd := exec.Command("nginx", "-s", "reload")
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (n *NginxPorts) reservePort() (int, bool) {
	if len(n.apps) == 100 {
		return 0, false
	}

	n.confMu.Lock()
	port := rand.Intn(100)

	// If the current port is in use, search for an open port linearly
	for n.ports[port] {
		port = port + 1%100
	}

	n.ports[port] = true
	n.confMu.Unlock()

	return port + 5000, true
}
