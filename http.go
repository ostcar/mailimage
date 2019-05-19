package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"

	"golang.org/x/xerrors"
)

//go:generate go run scripts/buildHTML.go
var indexTmpl = template.Must(template.New("indexPage").Parse(indexHTMLTemplate))

func serve(addr string) error {
	pool, err := newPool(redisAddr)
	if err != nil {
		return err
	}

	h := handler{redis: pool}

	http.Handle("/", errHandleFunc(h.index))
	http.Handle("/image/", errHandleFunc(h.image))
	http.Handle("/thumbnail/", errHandleFunc(h.thumbnail))
	http.Handle("/delete/", errHandleFunc(h.delete))

	return http.ListenAndServe(addr, nil)
}

type handler struct {
	redis *pool
}

// index returns the index page that list all images
func (h *handler) index(w http.ResponseWriter, r *http.Request) error {
	entries, err := h.redis.listEntries()
	if err != nil {
		return err
	}

	sort.Sort(sort.Reverse(byCreated(entries)))
	if err := indexTmpl.Execute(w, entries); err != nil {
		return xerrors.Errorf("can not execute index html template: %w", err)
	}
	return nil
}

// image returns an image via http.
func (h *handler) image(w http.ResponseWriter, r *http.Request) error {
	filename := r.URL.Path[len("/image/"):]
	requestedExt := filepath.Ext(filename)

	id, err := strconv.Atoi(filename[0 : len(filename)-len(requestedExt)])
	if err != nil {
		w.WriteHeader(404)
		return nil
	}

	dbExt, err := h.redis.getExtension(id)
	if err != nil {
		return err
	}

	if requestedExt != dbExt {
		w.WriteHeader(404)
		return nil
	}

	image, err := openImage(id, dbExt)
	if err != nil {
		return err
	}
	defer image.Close()

	if _, err := io.Copy(w, image); err != nil {
		return xerrors.Errorf("can not write image to response writer: %w", err)
	}
	return nil
}

// thumbnail returns a thunbmail from an image via http.
func (h *handler) thumbnail(w http.ResponseWriter, r *http.Request) error {
	filename := r.URL.Path[len("/thumbnail/"):]

	// valid thumbnails end with .jpg. Strip it to get the file number
	id, err := strconv.Atoi(filename[0 : len(filename)-4])
	if err != nil {
		w.WriteHeader(404)
		return nil
	}

	thumbnail, err := openThumbnail(id, h.redis)
	if err != nil {
		return err
	}
	defer thumbnail.Close()

	if _, err := io.Copy(w, thumbnail); err != nil {
		return xerrors.Errorf("can not write thumbnail to response writer: %w", err)
	}
	return nil
}

// delete deletes an image for an given token.
func (h *handler) delete(w http.ResponseWriter, r *http.Request) error {
	token := r.URL.Path[len("/delete/"):]

	if err := h.redis.deleteFromToken(token); err != nil {
		return err
	}

	http.Redirect(w, r, deleteRedirectURL, http.StatusFound)
	return nil
}

type errHandleFunc func(w http.ResponseWriter, r *http.Request) error

func (f errHandleFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := f(w, r); err != nil {
		if err == errUnknownImage {
			w.WriteHeader(404)
			return
		}

		log.Printf("Error: %v", err)
		http.Error(w, "Ups, something went wrong!", http.StatusInternalServerError)
	}
}
