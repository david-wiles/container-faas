package internal

import (
	"os"
	"os/exec"
	"text/template"
)

const nginxTemplate = "server {\n\tlisten {{ .Port }};\n  \n\tlocation / {\n\t\tproxy_pass {{ .Url }}/;\n\t}\n}"

func writeNginxConf(file string, port int, url string) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	t := template.Must(template.New("conf").Parse(nginxTemplate))
	err = t.Execute(f, struct {
		Port int
		Url  string
	}{
		port, url,
	})
	if err != nil {
		return err
	}

	return nil
}

func nginxReload() error {

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
