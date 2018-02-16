package admin

import (
	"html/template"
	"net/http"
)

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/menu.html",
		"control/templates/header.html",
	)
	t.ExecuteTemplate(w, "main", "")
}
