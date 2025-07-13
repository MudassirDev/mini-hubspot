package main

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

func RenderTemplate(w http.ResponseWriter, name string, data any) {
	cwd, _ := os.Getwd()
	layoutFiles, _ := filepath.Glob(cwd + "/frontend/templates/layouts/*.html")
	page := cwd + "/frontend/templates/pages/" + name + ".html"

	tmpl, err := template.ParseFiles(append(layoutFiles, page)...)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, "Render error", http.StatusInternalServerError)
	}
}
