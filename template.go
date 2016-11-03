package main

import (
	"errors"
	"html/template"
	"io"
	"path/filepath"
	"log"
)

var templates map[string]*template.Template

type page struct {
	base     string
	template string
	data     map[string]interface{}
}

func init() {
	err := initTemplates()

	if err != nil {
		log.Fatal(err)
	}
}

func initTemplates() error {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}

	templatesDir := "./templates/"

	layouts, err := filepath.Glob(templatesDir + "layouts/*.tmpl")

	if err != nil {
		return err
	}

	pages, err := filepath.Glob(templatesDir + "pages/*.tmpl")

	if err != nil {
		return err
	}

	for _, page := range pages {
		files := append(layouts, page)
		filename := filepath.Base(page)

		var err error

		templates[filename], err = template.New(filename).ParseFiles(files...)

		if err != nil {
			return err
		}
	}

	return nil
}

func renderTemplate(w io.Writer, name string, base string, data map[string]interface{}) error {
	if templates == nil {
		err := initTemplates()

		if err != nil {
			return err
		}
	}

	t, ok := templates[name]

	if !ok {
		return errors.New("unable to find template: " + name)
	}

	return t.ExecuteTemplate(w, base, data)
}
