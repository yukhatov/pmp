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

type responseParameterMapList struct {
	Maps           []*models.ParametersMaps
	PlatformTypeID uint64
}

type responseParameterMap struct {
	Item           models.ParametersMaps
	Success        bool
	IsEditing      bool
	Errors         []string
	Parameters     []*models.Parameter
	PlatformTypeID uint64
}

func GetParameterMapCreateHandler(w http.ResponseWriter, r *http.Request) {
	var item models.ParametersMaps
	errors := strings.Split(r.URL.Query().Get("error"), "|")
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/parameter_map/parameter_map.html"
	platformTypeID, err := getUintIDFromRequest(r, "platform_type_id")

	if err != nil || platformTypeID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/list/"), 302)
	}

	var parameters []*models.Parameter
	database.Postgres.Order("name").Find(&parameters)

	if successErr == nil {
		templateMain = "control/templates/parameter_map/parameter_map_success.html"
	}

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/parameter_map/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseParameterMap{
		Item:           item,
		Success:        success,
		IsEditing:      false,
		Errors:         errors,
		Parameters:     parameters,
		PlatformTypeID: platformTypeID,
	})
}

func GetParameterMapEditHandler(w http.ResponseWriter, r *http.Request) {
	var item models.ParametersMaps
	mapID, err := getUintIDFromRequest(r, "parameter_map_id")
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/parameter_map/parameter_map.html"
	platformTypeID, err := getUintIDFromRequest(r, "platform_type_id")

	if err != nil || platformTypeID == 0 || err != nil || mapID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/list/"), 302)
	}

	if successErr == nil {
		templateMain = "control/templates/parameter_map/parameter_map_success.html"
	}

	item.GetByID(mapID)

	var parameters []*models.Parameter
	database.Postgres.Order("name").Find(&parameters)

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/parameter_map/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseParameterMap{
		Item:           item,
		Success:        success,
		IsEditing:      true,
		Errors:         strings.Split(r.URL.Query().Get("error"), "|"),
		Parameters:     parameters,
		PlatformTypeID: platformTypeID,
	})
}

func PostParameterMapCreateHandler(w http.ResponseWriter, r *http.Request) {
	item := &models.ParametersMaps{}
	platformTypeID, _ := getUintIDFromRequest(r, "platform_type_id")

	r.ParseForm()
	item.PopulateData(r)

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/parameter_map/create/?error=%s", errorReplaced), 302)
	}

	if item.Create() {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/parameter_map/list/", platformTypeID), 302)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/parameter_map/%d/create/?success=false", platformTypeID), 302)
	}
}

func PostParameterMapEditHandler(w http.ResponseWriter, r *http.Request) {
	var mapID, _ = getUintIDFromRequest(r, "parameter_map_id")
	platformTypeID, _ := getUintIDFromRequest(r, "platform_type_id")
	item := &models.ParametersMaps{}

	r.ParseForm()
	item.GetByID(mapID)
	item.PopulateData(r)

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/parameter_map/%d/edit?error=%s", platformTypeID, item.ID, errorReplaced), 302)
	}

	if item.Save() {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/parameter_map/%d/edit/?success=true", platformTypeID, item.ID), 302)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/%d/parameter_map/%d/edit/?success=false", platformTypeID, item.ID), 302)
	}
}

func GetParamaterMapListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.ParametersMaps
	platformTypeID, err := getUintIDFromRequest(r, "platform_type_id")

	if err != nil || platformTypeID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/platform_type/list/"), 302)
	}

	database.Postgres.Preload("OriginalParameter").Where("advertiser_platform_id=?", platformTypeID).Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/parameter_map/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseParameterMapList{
		Maps:           list,
		PlatformTypeID: platformTypeID,
	})
}
