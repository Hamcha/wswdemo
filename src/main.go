package main

import (
	"flag"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
)

var demos *string
var root *string
var basepath *string

func truncate(str string) string {
	const size = 20
	if len(str) > size {
		return str[0:size] + "..."
	}
	return str
}

var funcMap = template.FuncMap{
	"truncate": truncate,
}

func webview(rw http.ResponseWriter, req *http.Request) {
	hometemplate := filepath.Join(filepath.Join(*root, "templates"), "index.html")
	t := template.Must(template.New("index.html").Funcs(funcMap).ParseFiles(hometemplate))

	demos, err := getDemos()
	if err != nil {
		http.Error(rw, "Cannot get demos: "+err.Error(), 500)
		return
	}

	sort.Sort(ByDate(demos))

	var duels []Demo
	var others []Demo

	for _, demo := range demos {
		if demo.IsDuel {
			duels = append(duels, demo)
		} else {
			others = append(others, demo)
		}
	}

	err = t.Execute(rw, struct {
		BaseURL string
		Duels   []Demo
		Others  []Demo
	}{
		*basepath,
		duels,
		others,
	})
	if err != nil {
		http.Error(rw, "Error while executing template: "+err.Error(), 500)
	}
}

func main() {
	bind := flag.String("bind", ":8353", "Address:Port to bind the webserver to")
	demos = flag.String("demofolder", "demos", "Folder containing the demos to serve")
	root = flag.String("root", ".", "Folder containing the static and templates folders")
	basepath = flag.String("basepath", "/", "Base URL from which the page is accessed")
	flag.Parse()

	http.Handle(*basepath+"static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(*root, "static")))))
	http.Handle(*basepath+"demos/", http.StripPrefix("/demos/", http.FileServer(http.Dir(*demos))))
	http.HandleFunc("/", webview)
	http.ListenAndServe(*bind, nil)
}
