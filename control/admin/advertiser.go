package admin

import (
	"fmt"
	"html/template"
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"

	"strings"

	"encoding/json"
	"strconv"

	"github.com/asaskevich/govalidator"
)

type responseAdvertiserEdit struct {
	Item      models.Advertiser
	IsEditing bool
	Success   bool
	Errors    []string
	CustomId  uint64
}

func AdvertiserListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.Advertiser
	database.Postgres.Preload("AdTags").Order("name").Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/advertiser/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", list)
}

func GetAdvertiserCreateHandler(w http.ResponseWriter, r *http.Request) {
	var advertiser models.Advertiser
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/advertiser/advertiser.html"
	errors := strings.Split(r.URL.Query().Get("error"), "|")

	if successErr == nil {
		templateMain = "control/templates/advertiser/advertiser_success.html"
	}

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		templateMain,
		"control/templates/advertiser/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseAdvertiserEdit{
		Item:      advertiser,
		Success:   success,
		Errors:    errors,
		IsEditing: false,
		CustomId:  FindAdvertNextCustomId(),
	})
}

func FindAdvertNextCustomId() uint64 {
	var ad models.Advertiser
	database.Postgres.Where("custom_id=(SELECT MAX(custom_id) FROM advertisers)").First(&ad)

	return ad.CustomID + 1
}

func GetAdvertiserEditHandler(w http.ResponseWriter, r *http.Request) {
	advertiserID, err := getUintIDFromRequest(r, "advertiser_id")
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/advertiser/advertiser.html"
	item := &models.Advertiser{}
	error := strings.Split(r.URL.Query().Get("error"), "|")

	if len(error) == 1 && error[0] == "" {
		error = nil
	}

	if successErr == nil {
		templateMain = "control/templates/advertiser/advertiser_success.html"
	}

	if err != nil || advertiserID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/advertiser/list/"), 302)
	}

	item.GetByID(advertiserID)

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/advertiser/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseAdvertiserEdit{
		Item:      *item,
		Success:   success,
		IsEditing: true,
		Errors:    error,
	})
}

func PostAdvertiserHandler(w http.ResponseWriter, r *http.Request) {
	advertiserID, advertiserIDErr := getUintIDFromRequest(r, "advertiser_id")
	isNewRecord := advertiserIDErr != nil || advertiserID == 0
	item := &models.Advertiser{}

	r.ParseForm()
	// TODO: add validation

	if isNewRecord {
		item.PopulateData(r)
	} else {
		item.GetByID(advertiserID)
		item.UpdateData(r)
	}

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
		if isNewRecord {
			http.Redirect(w, r, fmt.Sprintf("/advertiser/create/?error=%s", errorReplaced), 302)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/advertiser/%d/edit/?error=%s", item.ID, errorReplaced), 302)
		}
		return
	} else {
		if isNewRecord {
			if item.Create() {
				http.Redirect(w, r, "/advertiser/list/", 302)
			} else {
				http.Redirect(w, r, "/advertiser/create/?success=false", 302)
			}
		} else {
			isSaved, error := item.Save()

			if isSaved {
				http.Redirect(w, r, fmt.Sprintf("/advertiser/%d/edit/?success=true", item.ID), 302)
			} else {
				http.Redirect(w, r, fmt.Sprintf("/advertiser/%d/edit/?success=false&error=%s", item.ID, strings.Join(error, "|")), 302)
			}
		}
	}
}

func AdvertiserListJson(w http.ResponseWriter, r *http.Request) {
	response := getAdvertiserList()
	json_, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func AdvertiserListJsonByAdTag(w http.ResponseWriter, r *http.Request) {
	adTagID, err := getUintIDFromRequest(r, "ad_tag_id")

	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getAdvertiserListByAdTag(adTagID)
	json_, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func AdvertiserListJsonByPublisher(w http.ResponseWriter, r *http.Request) {
	publisherID, err := getUintIDFromRequest(r, "publisher_id")

	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getAdvertiserListByPublisher(publisherID)
	json_, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func getAdvertiserList() []responseSelectList {
	var response []responseSelectList
	items := []models.Advertiser{}

	database.Postgres.Order("name").Find(&items)
	response = make([]responseSelectList, len(items))

	for i, item := range items {
		response[i] = responseSelectList{
			ID:   item.ID,
			Name: item.Name,
		}
	}

	return response
}

func getAdvertiserListByPublisher(publisherID uint64) []responseSelectList {
	var response []responseSelectList
	items := []models.Advertiser{}

	database.Postgres.Model(&models.Advertiser{}).
		Joins("JOIN ad_tags ON ad_tags.advertiser_id=advertisers.id"+
			" JOIN ad_tag_publishers ON ad_tag_publishers.ad_tag_id=ad_tags.id").
		Where("ad_tag_publishers.publisher_id = ?", publisherID).
		Order("name").
		Group("advertisers.id").
		Find(&items)
	response = make([]responseSelectList, len(items))

	for i, item := range items {
		response[i] = responseSelectList{
			ID:   item.ID,
			Name: item.Name,
		}
	}

	return response
}

func getAdvertiserListByAdTag(adTagID uint64) []responseSelectList {
	var response []responseSelectList
	items := []models.Advertiser{}

	database.Postgres.Model(&models.Advertiser{}).
		Joins("JOIN ad_tags ON ad_tags.advertiser_id=advertisers.id").
		Where("ad_tags.id = ?", adTagID).
		Order("name").
		Group("advertisers.id").
		Find(&items)
	response = make([]responseSelectList, len(items))

	for i, item := range items {
		response[i] = responseSelectList{
			ID:   item.ID,
			Name: item.Name,
		}
	}

	return response
}
