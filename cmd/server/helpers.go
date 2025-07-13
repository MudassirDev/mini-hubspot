package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

func RenderTemplate(w http.ResponseWriter, name string, data any) {
	cwd, _ := os.Getwd()
	layoutFiles, _ := filepath.Glob(cwd + "/frontend/templates/layouts/*.html")
	page := cwd + "/frontend/templates/pages/" + name + ".html"
	fmt.Println(page)

	tmpl, err := template.ParseFiles(append(layoutFiles, page)...)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Render error", http.StatusInternalServerError)
	}
}
