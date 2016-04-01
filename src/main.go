package main

import (
	"flag"
	"html/template"
	"net/http"
	"path/filepath"
)

var demos *string
var templates *string

func truncate(str string) string {
	const size = 30
	if len(str) > size {
		return str[0:size] + "..."
	}
	return str
}

var funcMap = template.FuncMap{
	"truncate": truncate,
}

func webview(rw http.ResponseWriter, req *http.Request) {
	hometemplate := filepath.Join(*templates, "index.html")
	t := template.Must(template.New("index.html").Funcs(funcMap).ParseFiles(hometemplate))

	demos, err := getDemos()
	if err != nil {
		http.Error(rw, "Cannot get demos: "+err.Error(), 500)
		return
	}

	err = t.Execute(rw, struct {
		Demos []Demo
	}{
		demos,
	})
	if err != nil {
		http.Error(rw, "Error while executing template: "+err.Error(), 500)
	}
}

func main() {
	bind := flag.String("bind", ":8353", "Address:Port to bind the webserver to")
	demos = flag.String("demofolder", "demos", "Folder containing the demos to serve")
	templates = flag.String("templates", "templates", "Folder containing the HTML templates required")
	flag.Parse()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(*demos))))
	http.HandleFunc("/", webview)
	http.ListenAndServe(*bind, nil)
}
