package admin

import (
	"html/template"
	"net/http"

	"fmt"
	"strconv"
	"strings"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	"github.com/asaskevich/govalidator"
)

type responsePlatformType struct {
	Item      models.AdvertiserPlatformType
	Success   bool
	IsEditing bool
	Errors    []string
}

func GetPlatformTypeCreateHandler(w http.ResponseWriter, r *http.Request) {
	var item models.AdvertiserPlatformType
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/platform_type/platform_type.html"

	if successErr == nil {
		templateMain = "control/templates/platform_type/platform_type_success.html"
	}

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/platform_type/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePlatformType{
		Item:      item,
		Success:   success,
		IsEditing: false,
		Errors:    strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func GetPlatformTypeEditHandler(w http.ResponseWriter, r *http.Request) {
	var item models.AdvertiserPlatformType
	platformTypeID, err := getUintIDFromRequest(r, "platform_type_id")
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/platform_type/platform_type.html"

	if err != nil || platformTypeID == 0 || err != nil {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/list/"), 302)
	}

	if successErr == nil {
		templateMain = "control/templates/platform_type/platform_type_success.html"
	}

	item.GetByID(platformTypeID)

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/platform_type/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePlatformType{
		Item:      item,
		Success:   success,
		IsEditing: true,
		Errors:    strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func PostPlatformTypeCreateHandler(w http.ResponseWriter, r *http.Request) {
	item := &models.AdvertiserPlatformType{}

	r.ParseForm()
	item.PopulateData(r)

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/parameter_map/create/?error=%s", errorReplaced), 302)
	}

	if item.Create() {
		http.Redirect(w, r, "/platform_type/list/", 302)
	} else {
		http.Redirect(w, r, "/platform_type/create/?success=false", 302)
	}
}

func PostPlatformTypeEditHandler(w http.ResponseWriter, r *http.Request) {
	var platformTypeID, _ = getUintIDFromRequest(r, "platform_type_id")
	item := &models.AdvertiserPlatformType{}

	r.ParseForm()
	item.GetByID(platformTypeID)
	item.PopulateData(r)

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/edit?error=%s", platformTypeID, errorReplaced), 302)
	}

	if item.Save() {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/edit/?success=true", platformTypeID), 302)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/edit/?success=false", platformTypeID), 302)
	}
}

func GetPlatformTypeListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.AdvertiserPlatformType

	database.Postgres.Order("name").Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/platform_type/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", list)
}
