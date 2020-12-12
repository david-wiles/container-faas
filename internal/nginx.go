package internal

import (
	"os"
	"text/template"
)

const nginxTemplate = "server {\n\tlisten {{ .Port }};\n  \n\tlocation {\n\t\tproxy_pass {{ .Url }};\n\t}\n}"

func writeNginxConf(file string, port int, url string) error {
	f, err := os.Open(file)
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
