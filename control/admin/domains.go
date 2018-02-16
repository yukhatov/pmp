package admin

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/asaskevich/govalidator"

	"strings"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
)

type responseDomainsListEdit struct {
	DomainsList models.DomainsList
	IsEditing   bool
	Success     bool
	Errors      []string
}

func DomainsListHandler(w http.ResponseWriter, r *http.Request) {
	var list []models.DomainsList
	database.Postgres.Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/domains/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", list)
}

func DomainsListEditHandler(w http.ResponseWriter, r *http.Request) {
	var success bool
	domainsListID, err := getUintIDFromRequest(r, "domains_list_id")
	isNewRecord := err != nil || domainsListID == 0

	item := &models.DomainsList{}

	if r.Method == "POST" {
		r.ParseForm()

		if isNewRecord {
			item.PopulateData(r)
		} else {
			item.GetByID(domainsListID)
			item.UpdateData(r)
		}

		if _, err := govalidator.ValidateStruct(item); err != nil {
			errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
			if isNewRecord {
				http.Redirect(w, r, fmt.Sprintf("/domains/create/?error=%s", errorReplaced), 302)
			} else {
				http.Redirect(w, r, fmt.Sprintf("/domains/%d/edit/?error=%s", item.ID, errorReplaced), 302)
			}
			return
		} else {
			if isNewRecord {
				item.Create()
			} else {
				item.Save()
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/domains/%d/edit/?success=true", item.ID), 302)
		return
	} else {
		if isNewRecord {
			// TODO: error handling
			panic(err)
		}

		success = len(r.URL.Query().Get("success")) > 0

		item.GetByID(domainsListID)
	}

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/domains/edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseDomainsListEdit{
		DomainsList: *item,
		IsEditing:   true,
		Success:     success,
		Errors:      strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func DomainsListCreateHandler(w http.ResponseWriter, r *http.Request) {
	var domainsList models.DomainsList

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/domains/edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseDomainsListEdit{
		DomainsList: domainsList,
		Errors:      strings.Split(r.URL.Query().Get("error"), "|"),
	})
}
