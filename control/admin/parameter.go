package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	"github.com/asaskevich/govalidator"
)

type responseParameter struct {
	Item      models.Parameter
	Success   bool
	IsEditing bool
	Errors    []string
}

func GetParameterCreateHandler(w http.ResponseWriter, r *http.Request) {
	var item models.Parameter
	errors := strings.Split(r.URL.Query().Get("error"), "|")
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/parameter/parameter.html"

	if successErr == nil {
		templateMain = "control/templates/parameter/parameter_success.html"
	}

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/parameter/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseParameter{
		Item:      item,
		Success:   success,
		IsEditing: false,
		Errors:    errors,
	})
}

func GetParameterEditHandler(w http.ResponseWriter, r *http.Request) {
	var parameterID, err = getUintIDFromRequest(r, "parameter_id")
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/parameter/parameter.html"

	if successErr == nil {
		templateMain = "control/templates/parameter/parameter_success.html"
	}

	if err != nil || parameterID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/parameter/list/"), 302)
	}

	item := &models.Parameter{}
	item.GetByID(parameterID)

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/parameter/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseParameter{
		Item:      *item,
		Success:   success,
		IsEditing: true,
		Errors:    strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func PostParameterCreateHandler(w http.ResponseWriter, r *http.Request) {
	item := &models.Parameter{}

	r.ParseForm()
	item.PopulateData(r)

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/parameter/create/?error=%s", errorReplaced), 302)
	}

	if item.Create() {
		http.Redirect(w, r, "/parameter/list/", 302)
	} else {
		http.Redirect(w, r, "/publisher/create/?success=false", 302)
	}
}

func PostParameterEditHandler(w http.ResponseWriter, r *http.Request) {
	var parameterID, _ = getUintIDFromRequest(r, "parameter_id")
	item := &models.Parameter{}

	r.ParseForm()
	item.GetByID(parameterID)
	item.PopulateData(r)

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/parameter/%d/edit?error=%s", item.ID, errorReplaced), 302)
	}

	if item.Save() {
		http.Redirect(w, r, fmt.Sprintf("/parameter/%d/edit/?success=true", item.ID), 302)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/parameter/%d/edit/?success=false", item.ID), 302)
	}
}

func PostPrameterDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var parameterID, _ = getUintIDFromRequest(r, "parameter_id")

	item := &models.Parameter{}
	item.GetByID(parameterID)

	item.Delete()
}

func GetParamaterListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.Parameter

	database.Postgres.Order("name").Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/parameter/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", list)
}
