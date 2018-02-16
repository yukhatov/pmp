package admin

import (
	"html/template"
	"net/http"

	"fmt"

	"strings"

	"strconv"

	"encoding/json"

	"bitbucket.org/tapgerine/pmp/control/config"
	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	log "github.com/Sirupsen/logrus"
	"github.com/asaskevich/govalidator"
)

type responsePublisherEdit struct {
	Item      models.Publisher
	IsEditing bool
	Success   bool
	Errors    []string
	CustomId  uint64
}

type responsePublisherUserEdit struct {
	Item      models.User
	IsEditing bool
	Success   bool
	Errors    []string
}

type responsePublisherAdTagCreate struct {
	Item            models.AdTagPublisher
	Success         bool
	Publisher       models.Publisher
	AdvertisersList []models.Advertiser
	Errors          []string
}

type responsePublisherLinkCreate struct {
	Item         models.PublisherLink
	DomainsLists []models.DomainsList
	Success      bool
	IsEditing    bool
	Publisher    models.Publisher
	Errors       []string
}

type responsePublisherLinkAdTagPublisherCreate struct {
	PublisherLink      models.PublisherLink
	AdTagPublisherList []models.AdTagPublisher
	Success            bool
	Publisher          models.Publisher
	Errors             []string
}

type responsePublisherLinkAdTagCreate struct {
	PublisherLink models.PublisherLink
	AdTags        []models.AdTag
	Success       bool
	Publisher     models.Publisher
	Errors        []string
}

type responseSelectList struct {
	ID   uint64
	Name string
}

type responseSelectListString struct {
	ID   string
	Name string
}

type responseAddPublisherLinks struct {
	PublisherLinks []*models.PublisherLink
	AdTag          *models.AdTag
}

func PublisherListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.Publisher
	database.Postgres.Preload("AdTagPublisher").Preload("Links").Order("name").Find(&list)
	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/publisher/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", list)
}

func GetPublisherCreateHandler(w http.ResponseWriter, r *http.Request) {
	var item models.Publisher
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/publisher/publisher.html"
	errors := strings.Split(r.URL.Query().Get("error"), "|")

	if successErr == nil {
		templateMain = "control/templates/publisher/publisher_success.html"
	}

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/publisher/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePublisherEdit{
		Item:      item,
		Success:   success,
		IsEditing: false,
		Errors:    errors,
		CustomId:  FindPublisherNextCustomId(),
	})
}

func FindPublisherNextCustomId() uint64 {
	var ad models.Publisher
	database.Postgres.Where("custom_id=(SELECT MAX(custom_id) FROM publishers)").First(&ad)

	return ad.CustomID + 1
}

func GetPublisherEditHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, err = getUintIDFromRequest(r, "publisher_id")
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/publisher/publisher.html"
	error := strings.Split(r.URL.Query().Get("error"), "|")

	if len(error) == 1 && error[0] == "" {
		error = nil
	}

	if successErr == nil {
		templateMain = "control/templates/publisher/publisher_success.html"
	}

	if err != nil || publisherID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/publisher/list/"), 302)
	}

	item := &models.Publisher{}
	item.GetByID(publisherID)

	item.CheckIfTargetingIDExists()

	if item.TargetingLink == "" && item.TargetingPrice > 0.0 {
		var parameters []models.Parameter
		database.Postgres.Find(&parameters)
		link := fmt.Sprintf("https://%s/rotator/target?pub=%s", config.RotatorDomain, item.TargetingID)
		for _, parameter := range parameters {
			link += fmt.Sprintf("&%s=%s", parameter.Shortcut, parameter.Macros)
		}
		link += fmt.Sprintf("&price=%.2f", item.TargetingPrice)
		item.TargetingLink = link
		item.Save()
	}

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/publisher/form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePublisherEdit{
		Item:      *item,
		Success:   success,
		IsEditing: true,
		Errors:    error,
	})
}

func PostPublisherHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, publisherIDErr = getUintIDFromRequest(r, "publisher_id")
	isNewRecord := publisherIDErr != nil || publisherID == 0
	item := &models.Publisher{}

	r.ParseForm()

	if isNewRecord {
		item.PopulateData(r)
	} else {
		item.GetByID(publisherID)
		item.UpdateData(r)
	}

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		if isNewRecord {
			http.Redirect(w, r, fmt.Sprintf("/publisher/create/?error=%s", errorReplaced), 302)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/publisher/%d/edit/?error=%s", item.ID, errorReplaced), 302)
		}
		return
	} else {
		if isNewRecord {
			if item.Create() {
				http.Redirect(w, r, "/publisher/list/", 302)
			} else {
				http.Redirect(w, r, "/publisher/create/?success=false", 302)
			}
		} else {
			isSaved, error := item.Save()

			if isSaved {
				http.Redirect(w, r, fmt.Sprintf("/publisher/%d/edit/?success=true", item.ID), 302)
			} else {
				http.Redirect(w, r, fmt.Sprintf("/publisher/%d/edit/?success=false&error=%s", item.ID, strings.Join(error, "|")), 302)
			}
		}
	}
}

func PublisherAdTagsListHandler(w http.ResponseWriter, r *http.Request) {
	publisherID, publisherIDErr := getUintIDFromRequest(r, "publisher_id")
	if publisherIDErr != nil {
		log.WithError(publisherIDErr).Warn("no publisher found")
		return
		//panic(publisherIDErr)
	}

	item := models.Publisher{}

	database.Postgres.Preload("AdTagPublisher").Preload("AdTagPublisher.AdTag").Where("id = ?", publisherID).Find(&item)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/publisher/ad_tags.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", item)
}

func PublisherUserEditHandler(w http.ResponseWriter, r *http.Request) {
	publisherID, _ := getUintIDFromRequest(r, "publisher_id")
	success := len(r.URL.Query().Get("success")) > 0

	publisher := &models.Publisher{}
	publisher.GetByID(publisherID)

	item := &models.User{}
	isNewRecord := publisher.UserID == 0
	isEditing := !isNewRecord

	if r.Method == "POST" {
		r.ParseForm()

		if isNewRecord {
			item.PopulateData(r)
			item.PublisherID = publisher.ID
			item.Role = "publisher"
		} else {
			item.GetByID(publisher.UserID)
			item.UpdateData(r)
		}

		if _, err := govalidator.ValidateStruct(item); err != nil {
			errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
			http.Redirect(w, r, fmt.Sprintf("/publisher/%d/user/edit/?error=%s", publisher.ID, errorReplaced), 302)
			return
		} else {
			success = true
			if isNewRecord {
				item.Create()
				publisher.UserID = item.ID
				publisher.Save()
			} else {
				item.Save()
			}
		}
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/user/edit/?success=true", publisher.ID), 302)
		return
	}
	if !isNewRecord {
		item.GetByID(publisher.UserID)
	}

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/publisher/edit_user.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePublisherUserEdit{
		Item:      *item,
		IsEditing: isEditing,
		Success:   success,
	})
}

func GetPublisherAdTagCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, publisherIDErr = getUintIDFromRequest(r, "publisher_id")
	var item models.AdTagPublisher
	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/publisher/ad_tag.html"

	if successErr == nil {
		templateMain = "control/templates/publisher/ad_tag_success.html"
	}

	if publisherIDErr != nil {
		log.WithError(publisherIDErr).Warn("no publisher found")
		http.Redirect(w, r, "/publisher/list/", 302)
	}

	publisher := &models.Publisher{}
	publisher.GetByID(publisherID)

	if publisher == nil {
		http.Redirect(w, r, fmt.Sprintf("/publisher/list/"), 302)
	}

	advertisersList := []models.Advertiser{}
	database.Postgres.Order("name").Find(&advertisersList)

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/publisher/ad_tag_form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePublisherAdTagCreate{
		Item:            item,
		Publisher:       *publisher,
		AdvertisersList: advertisersList,
		Errors:          strings.Split(r.URL.Query().Get("error"), "|"),
		Success:         success,
	})
}

func PostPublisherAdTagCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, publisherIDErr = getUintIDFromRequest(r, "publisher_id")
	item := &models.AdTagPublisher{}
	publisher := &models.Publisher{}
	adTag := &models.AdTag{}

	if publisherIDErr != nil {
		http.Redirect(w, r, "/publisher/list/", 302)
	}

	r.ParseForm()

	publisher.GetByIDFromRequest(r)
	adTag.GetByIDFromForm(r, "ad_tag")

	item.PopulateData(r)
	item.PublisherID = publisherID
	item.Publisher = *publisher
	item.AdTagID = adTag.ID
	item.AdTag = *adTag
	/*item.GenerateName()*/

	govalidator.CustomTypeTagMap.Set("priceCompareValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
		price := i.(float64)
		return price < adTag.Price
	}))

	govalidator.CustomTypeTagMap.Set("priceRangeValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
		price := i.(float64)
		return 50.0 >= price && price >= 1.0
	}))

	//govalidator.CustomTypeTagMap.Set("shaveValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
	//	shave := i.(uint64)
	//	return 99 >= shave && shave >= 0
	//}))

	govalidator.CustomTypeTagMap.Set("samePubValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
		adTagPub := context.(models.AdTagPublisher)
		var count int
		database.Postgres.
			Model(&models.AdTagPublisher{}).
			Where(&models.AdTagPublisher{
				AdTagID:     adTagPub.AdTagID,
				Price:       adTagPub.Price,
				PublisherID: adTagPub.PublisherID,
			}).
			Where("id != ?", adTagPub.ID).
			Count(&count)

		return count == 0
	}))

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/ad_tag/create/?success=false&error=%s", publisherID, errorReplaced), 302)
	} else {
		if item.Create() {
			http.Redirect(w, r, fmt.Sprintf("/publisher/%d/ad_tag/list/", publisherID), 302)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/publisher/%d/ad_tag/create/?success=false", publisherID), 302)
		}
	}
}

func LogAsPublisherHandler(w http.ResponseWriter, r *http.Request) {
	publisherID, _ := getUintIDFromRequest(r, "publisher_id")

	sessionCookie, _ := r.Cookie("Session")
	session := &models.Session{}
	session.GetByID(sessionCookie.Value)

	session.User.PublisherID = publisherID
	session.User.Save()
	http.Redirect(w, r, "/publisher_admin/", 302)
}

func PublisherListJsonByAdTag(w http.ResponseWriter, r *http.Request) {
	adTagID, err := getUintIDFromRequest(r, "ad_tag_id")

	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getPublishersListByAdTag(adTagID)
	json_, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func PublisherLinksListJson(w http.ResponseWriter, r *http.Request) {
	publisherID, err := getUintIDFromRequest(r, "publisher_id")

	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getPublishersLinksList(publisherID)
	json_, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func PublisherListJsonByAdvertiser(w http.ResponseWriter, r *http.Request) {
	advertiserID, err := getUintIDFromRequest(r, "advertiser_id")

	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getPublishersListByAdvertiser(advertiserID)
	json_, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func PublisherListJson(w http.ResponseWriter, r *http.Request) {
	response := getPublishersList()
	json_, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func getPublishersList() []responseSelectList {
	var response []responseSelectList
	items := []models.Publisher{}

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

func getPublishersListByAdvertiser(advertiserID uint64) []responseSelectList {
	var response []responseSelectList
	items := []models.Publisher{}

	database.Postgres.Model(&models.Publisher{}).
		Joins("JOIN ad_tag_publishers ON publishers.id=ad_tag_publishers.publisher_id"+
			" JOIN ad_tags ON ad_tag_publishers.ad_tag_id=ad_tags.id").
		Where("ad_tags.advertiser_id = ?", advertiserID).
		Group("publishers.id").
		Order("name").
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

func getPublishersLinksList(publisherID uint64) []responseSelectListString {
	var response []responseSelectListString
	items := []models.PublisherLink{}

	database.Postgres.Model(&models.PublisherLink{}).
		Where("publisher_id = ?", publisherID).
		Order("name").
		Find(&items)
	response = make([]responseSelectListString, len(items))

	for i, item := range items {
		response[i] = responseSelectListString{
			ID:   item.ID,
			Name: item.Name,
		}
	}

	return response
}

func getPublishersListByAdTag(adTagID uint64) []responseSelectList {
	var response []responseSelectList
	items := []models.Publisher{}

	database.Postgres.Model(&models.Publisher{}).
		Joins("JOIN ad_tag_publishers ON publishers.id=ad_tag_publishers.publisher_id").
		Group("publishers.id").
		Where("ad_tag_publishers.ad_tag_id = ?", adTagID).
		Order("name").
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

func PublisherLinksListHandler(w http.ResponseWriter, r *http.Request) {
	var publisher models.Publisher
	publisher.GetByIDFromRequest(r)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/publisher/link_list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", publisher)
}

func GetPublisherLinkCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, publisherIDErr = getUintIDFromRequest(r, "publisher_id")

	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/publisher/link.html"

	if successErr == nil {
		templateMain = "control/templates/publisher/link_success.html"
	}

	if publisherIDErr != nil {
		log.WithError(publisherIDErr).Warn("no publisher found")
		http.Redirect(w, r, "/publisher/list/", 302)
	}

	publisher := &models.Publisher{}
	publisher.GetByID(publisherID)

	if publisher == nil {
		http.Redirect(w, r, fmt.Sprintf("/publisher/list/"), 302)
	}

	platform := r.URL.Query().Get("platform")

	var parameters []models.Parameter
	database.Postgres.Where("platform = ?", platform).Find(&parameters)

	item := &models.PublisherLink{}
	item.GenerateID()
	item.Platform = platform
	link := fmt.Sprintf("https://%s/rotator/target/v2?pub=%s", config.RotatorDomain, item.ID)
	for _, parameter := range parameters {
		link += fmt.Sprintf("&%s=%s", parameter.Shortcut, parameter.Macros)
	}
	link += fmt.Sprintf("&price=%s", "[PRICE]")
	link += fmt.Sprintf("&response=%s", "[RESPONSE:vast20wrapper|vast20vpaid]")
	item.Link = link

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/publisher/link_form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	var domainsLists []models.DomainsList
	database.Postgres.Find(&domainsLists)

	t.ExecuteTemplate(w, "main", responsePublisherLinkCreate{
		Item:         *item,
		DomainsLists: domainsLists,
		Publisher:    *publisher,
		Errors:       strings.Split(r.URL.Query().Get("error"), "|"),
		Success:      success,
	})
}

func PostPublisherLinkCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, _ = getUintIDFromRequest(r, "publisher_id")
	item := &models.PublisherLink{}

	r.ParseForm()
	item.PopulateData(r)
	item.PublisherID = publisherID

	govalidator.CustomTypeTagMap.Set("priceValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
		price := i.(float64)

		return price >= 0.01
	}))

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)

		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/create/?success=false&error=%s", publisherID, errorReplaced), 302)
		return
	} else {
		if item.Create() {
			http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/list/", publisherID), 302)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/create/?success=false", publisherID), 302)
		}
	}
}

func GetPublisherLinkEditHandler(w http.ResponseWriter, r *http.Request) {
	var linkID, linkIDERR = getStringFromRequest(r, "link_id")

	success, successErr := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/publisher/link.html"
	error := strings.Split(r.URL.Query().Get("error"), "|")

	if len(error) == 1 && error[0] == "" {
		error = nil
	}

	if successErr == nil {
		templateMain = "control/templates/publisher/link_success.html"
	}

	if linkIDERR != nil {
		log.WithError(linkIDERR).Warn("no link found")
		http.Redirect(w, r, "/publisher/list/", 302)
	}

	item := &models.PublisherLink{}
	item.GetByID(linkID)

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/publisher/link_form.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	var domainsLists []models.DomainsList
	database.Postgres.Find(&domainsLists)

	t.ExecuteTemplate(w, "main", responsePublisherLinkCreate{
		Item:         *item,
		DomainsLists: domainsLists,
		IsEditing:    true,
		Publisher:    item.Publisher,
		Errors:       error,
		Success:      success,
	})
}

func PostPublisherLinkEditHandler(w http.ResponseWriter, r *http.Request) {
	var linkID, _ = getStringFromRequest(r, "link_id")
	item := &models.PublisherLink{}

	r.ParseForm()
	item.GetByID(linkID)
	item.UpdateData(r)

	govalidator.CustomTypeTagMap.Set("priceValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
		price := i.(float64)

		return price >= 0.01
	}))

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/%s/edit/?success=false&error=%s", item.PublisherID, item.ID, errorReplaced), 302)

		return
	}

	if item.Save() {
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/%s/edit/?success=true", item.PublisherID, item.ID), 302)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/%s/edit/?success=false", item.PublisherID, item.ID), 302)
	}
}

func PublisherLinksAdTagPublisherListHandler(w http.ResponseWriter, r *http.Request) {
	linkID, _ := getStringFromRequest(r, "link_id")
	link := &models.PublisherLink{}
	link.GetByID(linkID)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/publisher/link_ad_tag_publisher_list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", link)
}

func GetPublisherLinkAdTagCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, _ = getUintIDFromRequest(r, "publisher_id")
	var linkID, _ = getStringFromRequest(r, "link_id")

	success, _ := strconv.ParseBool(r.URL.Query().Get("success"))

	publisher := &models.Publisher{}
	publisher.GetByID(publisherID)

	if publisher == nil {
		http.Redirect(w, r, fmt.Sprintf("/publisher/list/"), 302)
	}

	publisherLink := &models.PublisherLink{}
	publisherLink.GetByID(linkID)

	var adTagsList []models.AdTag
	database.Postgres.Where("is_archived = false AND minimum_margin > 0 AND ad_tags.price - minimum_margin >= ?", publisherLink.Price).Find(&adTagsList)

	t, _ := template.ParseFiles(
		"control/templates/publisher/link_ad_tag_add_form.html",
		"control/templates/main.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePublisherLinkAdTagCreate{
		PublisherLink: *publisherLink,
		Publisher:     *publisher,
		AdTags:        adTagsList,
		Errors:        strings.Split(r.URL.Query().Get("error"), "|"),
		Success:       success,
	})

}

func GetPublisherLinkAdTagPublisherCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, _ = getUintIDFromRequest(r, "publisher_id")
	var linkID, _ = getStringFromRequest(r, "link_id")

	success, _ := strconv.ParseBool(r.URL.Query().Get("success"))
	templateMain := "control/templates/publisher/link_ad_tag_publisher_add_form.html"

	publisher := &models.Publisher{}
	publisher.GetByID(publisherID)

	if publisher == nil {
		http.Redirect(w, r, fmt.Sprintf("/publisher/list/"), 302)
	}

	publisherLink := &models.PublisherLink{}
	publisherLink.GetByID(linkID)

	adTagPublisherList := []models.AdTagPublisher{}
	database.Postgres.Where("publisher_id = ?", publisherLink.PublisherID).Preload("AdTag").Order("name").Find(&adTagPublisherList)

	t, _ := template.ParseFiles(
		templateMain,
		"control/templates/main.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responsePublisherLinkAdTagPublisherCreate{
		PublisherLink:      *publisherLink,
		Publisher:          *publisher,
		AdTagPublisherList: adTagPublisherList,
		Errors:             strings.Split(r.URL.Query().Get("error"), "|"),
		Success:            success,
	})
}

func PostPublisherLinkAdTagPublisherCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, _ = getUintIDFromRequest(r, "publisher_id")
	var linkID, _ = getStringFromRequest(r, "link_id")
	item := &models.PublisherLinkAdTagPublisher{}

	r.ParseForm()
	item.PopulateData(r)
	item.PublisherLinkID = linkID
	item.IsActive = true

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/%d/add_ad_tag_publisher/?error=%s", publisherID, linkID, errorReplaced), 302)
		return
	} else {
		item.Create()
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/%s/list/", publisherID, linkID), 302)
	}
}

func PostPublisherLinkAdTagCreateHandler(w http.ResponseWriter, r *http.Request) {
	var publisherID, _ = getUintIDFromRequest(r, "publisher_id")
	var linkID, _ = getStringFromRequest(r, "link_id")

	r.ParseForm()

	adTagID, _ := strconv.ParseUint(r.Form.Get("ad_tag_id"), 10, 0)

	var adTag models.AdTag
	adTag.GetByID(adTagID, r.Context().Value("userID").(uint64))

	var adTagPublisher models.AdTagPublisher
	adTagPublisher.AdTagID = adTagID
	adTagPublisher.Price = adTag.Price - adTag.MinimumMargin
	adTagPublisher.PublisherID = publisherID
	adTagPublisher.IsActive = true
	adTagPublisher.IsLocked = true
	adTagPublisher.Create()

	item := &models.PublisherLinkAdTagPublisher{}
	item.AdTagPublisherID = adTagPublisher.ID
	item.PublisherLinkID = linkID
	item.IsActive = true

	if _, err := govalidator.ValidateStruct(item); err != nil {
		errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/%d/add_tag/?error=%s", publisherID, linkID, errorReplaced), 302)
		return
	} else {
		item.Create()
		http.Redirect(w, r, fmt.Sprintf("/publisher/%d/link/%s/list/", publisherID, linkID), 302)
	}
}

func PublisherLinkAdTagActivationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}

	item := &models.PublisherLinkAdTagPublisher{}
	item.GetByID(id)
	item.IsActive = !item.IsActive
	item.Save()

	w.Write([]byte("success"))
}

func AdTagPublisherLinkListHandler(w http.ResponseWriter, r *http.Request) {
	adTagID, err := getUintIDFromRequest(r, "ad_tag_id")
	var items []*models.PublisherLink

	if err != nil {
		http.Redirect(w, r, "/advertiser/list/", 302)
	}

	adTag := &models.AdTag{}
	adTag.GetByID(adTagID, r.Context().Value("userID").(uint64))

	var deviceType string

	if adTag.DeviceType == "mobile" {
		deviceType = "desktop"
	} else {
		deviceType = adTag.DeviceType
	}

	database.Postgres.Model(&models.PublisherLink{}).
		Select("*, publisher_link_id AS has_ad_tag").
		Joins(fmt.Sprintf(`LEFT JOIN (
			SELECT publisher_link_id FROM publisher_link_ad_tag_publishers
			LEFT JOIN ad_tag_publishers ON ad_tag_publishers.id = publisher_link_ad_tag_publishers.ad_tag_publisher_id
			WHERE ad_tag_id = %d AND publisher_link_ad_tag_publishers.is_active IS TRUE
			GROUP BY publisher_link_id
			) AS platp ON publisher_links.id = platp.publisher_link_id`, adTagID)).
		Where("(price <= ? OR publisher_link_id IS NOT NULL) AND platform = ?", adTag.Price-adTag.MinimumMargin, deviceType).
		Group("publisher_link_id, publisher_links.id").
		Order("has_ad_tag").
		Find(&items)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/publisher_link_list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseAddPublisherLinks{
		PublisherLinks: items,
		AdTag:          adTag,
	})
}

func AdTagPublisherLinkConnect(w http.ResponseWriter, r *http.Request) {
	adTagID, _ := getUintIDFromRequest(r, "ad_tag_id")
	var publisherLinkID, _ = getStringFromRequest(r, "publisher_link_id")

	publisherLink := &models.PublisherLink{}
	publisherLink.GetByID(publisherLinkID)
	publisherID := publisherLink.PublisherID
	var adTagPublisher = &models.AdTagPublisher{}

	database.Postgres.Where("ad_tag_id = ? AND publisher_id = ?", adTagID, publisherID).Find(adTagPublisher)

	if adTagPublisher.ID != "" {
		publisherLinkAdTagPublisher := &models.PublisherLinkAdTagPublisher{}
		database.Postgres.Where("ad_tag_publisher_id = ? AND publisher_link_id = ?", adTagPublisher.ID, publisherLinkID).Find(publisherLinkAdTagPublisher)

		if publisherLinkAdTagPublisher.ID != 0 {
			publisherLinkAdTagPublisher.IsActive = true
			publisherLinkAdTagPublisher.Save()
		} else {
			publisherLinkAdTagPublisher.IsActive = true
			publisherLinkAdTagPublisher.AdTagPublisherID = adTagPublisher.ID
			publisherLinkAdTagPublisher.PublisherLinkID = publisherLinkID
			publisherLinkAdTagPublisher.Create()
		}
	} else {
		adTagPublisher.AdTagID = adTagID
		adTagPublisher.Price = publisherLink.Price
		adTagPublisher.IsActive = true
		adTagPublisher.PublisherID = publisherID
		adTagPublisher.CurrentUserID = r.Context().Value("userID").(uint64)

		if adTagPublisher.Create() {
			publisherLinkAdTagPublisher := &models.PublisherLinkAdTagPublisher{}
			publisherLinkAdTagPublisher.AdTagPublisherID = adTagPublisher.ID
			publisherLinkAdTagPublisher.PublisherLinkID = publisherLinkID
			publisherLinkAdTagPublisher.IsActive = true

			publisherLinkAdTagPublisher.Create()
		}
	}
}

func AdTagPublisherLinkDisconnect(w http.ResponseWriter, r *http.Request) {
	var publisherLinkID, _ = getStringFromRequest(r, "publisher_link_id")
	var adTagID, _ = getStringFromRequest(r, "ad_tag_id")

	publisherLink := &models.PublisherLink{}
	publisherLink.GetByID(publisherLinkID)
	var adTagPublisher = &models.AdTagPublisher{}

	database.Postgres.Where("ad_tag_id = ? AND publisher_id = ?", adTagID, publisherLink.PublisherID).Find(adTagPublisher)

	if adTagPublisher.ID != "0" {
		publisherLinkAdTagPublisher := &models.PublisherLinkAdTagPublisher{}
		database.Postgres.Where("ad_tag_publisher_id = ? AND publisher_link_id = ?", adTagPublisher.ID, publisherLinkID).Find(publisherLinkAdTagPublisher)

		if publisherLinkAdTagPublisher.ID != 0 {
			publisherLinkAdTagPublisher.IsActive = false
			publisherLinkAdTagPublisher.Save()
		}
	}
}
