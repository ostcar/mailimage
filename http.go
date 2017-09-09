package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
)

func serveImage(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/image/"):]
	requestedExtension := filepath.Ext(filename)
	id, err := strconv.Atoi(filename[0 : len(filename)-len(requestedExtension)])
	if err != nil {
		w.WriteHeader(404)
		return
	}
	image, dbExtension, err := getImage(id)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	if requestedExtension != dbExtension {
		w.WriteHeader(404)
		return
	}
	w.Write(image)
}

func serveThumbnail(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/thumbnail/"):]
	id, err := strconv.Atoi(filename[0 : len(filename)-4])
	if err != nil {
		w.WriteHeader(404)
		return
	}
	thumbnail, err := getThumbnail(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(thumbnail)
}

//go:generate go run scripts/buildHTML.go
func servePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("").Parse(indexHTMLTemplate)
	if err != nil {
		http.Error(w, fmt.Sprintf("can not parse template: %s", err), 500)
		return
	}
	entries, err := listEntries()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	sort.Sort(sort.Reverse(ByCreated(entries)))
	tmpl.Execute(w, entries)
}

func serveDelete(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Path[len("/delete/"):]
	err := deleteFromToken(token)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, deleteRedirectURL, http.StatusFound)
}

func serve(dev string) {
	http.HandleFunc("/", servePage)
	http.HandleFunc("/image/", serveImage)
	http.HandleFunc("/thumbnail/", serveThumbnail)
	http.HandleFunc("/delete/", serveDelete)

	log.Fatal(http.ListenAndServe(dev, nil))
}
