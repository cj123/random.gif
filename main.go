package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"time"
	"log"
	"math/rand"
	"github.com/dhowett/gotimeout"
)

type gifURL string

type handler struct {
	store gifStore
	expirables *gotimeout.Map
}

func main() {
	store := newDiskStore("./gifs")
	err := store.Init()

	if err != nil {
		log.Fatal(err)
	}

	handler := handler{
		store: store,
		expirables: gotimeout.NewMap(),
	}

	r := mux.NewRouter()

	r.HandleFunc("/", handler.indexHandler)
	r.HandleFunc("/all", handler.allHandler).Methods("GET")
	r.HandleFunc("/submit", handler.submitHandler).Methods("POST")
	r.HandleFunc("/random", handler.randomHandler).Methods("GET")
	r.HandleFunc("/gif/{id}", handler.individualHandler).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func (h *handler) indexHandler(w http.ResponseWriter, r *http.Request) {
	err := renderTemplate(w, "index.tmpl", "layout.tmpl", nil)

	if err != nil {
		log.Fatal(err)
		http.Error(w, "bad template", http.StatusInternalServerError)
	}
}

func (h *handler) submitHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	url, ok := r.Form["url"]

	if !ok || len(url) != 1 {
		http.Error(w, "bad url", http.StatusBadRequest)
		return
	}

	// download url and store it
	gURL := gifURL(url[0])

	b, err := download(gURL)

	if err != nil {
		http.Error(w, "can't download gif. are you sure you gave me an url?", http.StatusBadRequest)
		return
	}

	err = h.store.Store(b, gURL)

	if err == gifAlreadyExistsError {
		http.Error(w, "we've already got that gif, thanks!", http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, "can't save gif :( " + err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (h *handler) randomHandler(w http.ResponseWriter, r *http.Request) {
	var hash string

	if val, ok := h.expirables.Get("gif-key"); ok {
		hash = val.(string)
	} else {
		all := h.store.All()
		keys := make([]string, 0, len(all))

		for key, _ := range all {
			keys = append(keys, key)
		}

		hash = keys[rand.Intn(len(all))]
		h.expirables.Put("gif-key", hash, 5 * time.Minute)
	}

	gif, err := h.store.Get(hash)

	if err != nil || gif == nil  {
		http.Error(w, "we could not find the gif you seek", http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "image/gif")
	w.Write(gif)
}

func (h *handler) individualHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]

	gif, err := h.store.Get(id)

	if err != nil || gif == nil  {
		http.Error(w, "we could not find the gif you seek", http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "image/gif")
	w.Write(gif)
}

func (h *handler) allHandler(w http.ResponseWriter, r *http.Request) {
	err := renderTemplate(w, "all.tmpl", "layout.tmpl", map[string]interface{}{
		"gifs": h.store.All(),
	})

	if err != nil {
		log.Fatal(err)
		http.Error(w, "bad template", http.StatusInternalServerError)
	}
}
