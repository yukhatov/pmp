package admin

import (
	"fmt"
	"html/template"
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"

	"strings"

	"time"

	"encoding/json"

	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/asaskevich/govalidator"
	"github.com/roistat/go-clickhouse"
)

type responseEdit struct {
	AdvertiserID   uint64
	AdTag          models.AdTag
	IsEditing      bool
	Success        bool
	Errors         []string
	GeoCountryList []GeoCountry
	Types          []*models.AdvertiserPlatformTypes
	DomainsLists   []models.DomainsList
}

type responseList struct {
	AdTags       []*models.AdTag
	Advertiser   models.Advertiser
	TodayDate    string
	ShowArchived bool
}

type publishersListForAdTag struct {
	AdTagPubID string
	Name       string
}

func AdTagsListHandler(w http.ResponseWriter, r *http.Request) {
	advertiserID, err := getUintIDFromRequest(r, "advertiser_id")

	showArchived := r.URL.Query().Get("show_archived") == "true"

	if err != nil {
		// TODO: add error handling
		panic(err)
	}

	var list []*models.AdTag

	if showArchived {
		database.Postgres.
			Where("advertiser_id = ? AND is_archived = true", advertiserID).
			Order("updated_at desc").
			Find(&list)
	} else {
		database.Postgres.
			Where("advertiser_id = ? AND (is_archived = false OR is_archived is null)", advertiserID).
			Order("updated_at desc").
			Find(&list)
	}

	var advertiser models.Advertiser
	advertiser.ID = advertiserID
	database.Postgres.First(&advertiser)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseList{
		AdTags:       list,
		Advertiser:   advertiser,
		TodayDate:    time.Now().Format("2006-01-02"),
		ShowArchived: showArchived,
	})
}

func AdTagEditHandler(w http.ResponseWriter, r *http.Request) {
	var success bool
	adTagID, adTagIDErr := getUintIDFromRequest(r, "ad_tag_id")
	isNewRecord := adTagIDErr != nil || adTagID == 0

	var types []*models.AdvertiserPlatformTypes
	database.Postgres.Find(&types)

	item := &models.AdTag{}

	if r.Method == "POST" {
		r.ParseForm()

		if isNewRecord {
			item.PopulateData(r)
		} else {
			item.GetByID(adTagID, r.Context().Value("userID").(uint64))
			item.UpdateData(r)
		}

		item.Advertiser.ID = item.AdvertiserID
		database.Postgres.First(&item.Advertiser)

		govalidator.CustomTypeTagMap.Set("priceValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
			price := i.(float64)

			return price >= 0.01
		}))
		govalidator.CustomTypeTagMap.Set("marginValidator", govalidator.CustomTypeValidator(func(i interface{}, context interface{}) bool {
			return context.(models.AdTag).Price >= context.(models.AdTag).MinimumMargin

		}))

		if _, err := govalidator.ValidateStruct(item); err != nil {
			errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
			if isNewRecord {
				http.Redirect(w, r, fmt.Sprintf("/ad_tag/create/%d?error=%s", item.AdvertiserID, errorReplaced), 302)
			} else {
				http.Redirect(w, r, fmt.Sprintf("/ad_tag/%d/edit/?error=%s", item.ID, errorReplaced), 302)
			}
			return
		} else {
			if isNewRecord {
				item.Create()
			} else {
				item.Save()
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/ad_tag/%d/edit/?success=true", item.ID), 302)
		return

	} else {
		if isNewRecord {
			// TODO: error handling
			panic(adTagIDErr)
		}

		success = len(r.URL.Query().Get("success")) > 0

		item.GetByID(adTagID, r.Context().Value("userID").(uint64))
	}

	var domainsLists []models.DomainsList
	database.Postgres.Find(&domainsLists)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseEdit{
		AdvertiserID:   item.AdvertiserID,
		AdTag:          *item,
		IsEditing:      true,
		Success:        success,
		Errors:         strings.Split(r.URL.Query().Get("error"), "|"),
		GeoCountryList: GeoCountryList,
		Types:          types,
		DomainsLists:   domainsLists,
	})
}

func AdTagCreateHandler(w http.ResponseWriter, r *http.Request) {
	var adTag models.AdTag
	advertiserID, err := getUintIDFromRequest(r, "advertiser_id")

	if err != nil {
		// TODO: add error handling
		panic(err)
	}

	var types []*models.AdvertiserPlatformTypes
	database.Postgres.Find(&types)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseEdit{
		AdvertiserID:   advertiserID,
		AdTag:          adTag,
		Errors:         strings.Split(r.URL.Query().Get("error"), "|"),
		GeoCountryList: GeoCountryList,
		Types:          types,
	})
}

func AdTagActivationHandler(w http.ResponseWriter, r *http.Request) {
	adTagID, adTagIDErr := getUintIDFromRequest(r, "ad_tag_id")
	if adTagIDErr != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.AdTag{}
	item.GetByID(adTagID, r.Context().Value("userID").(uint64))

	item.IsActive = !item.IsActive

	if item.IsActive && !item.IsLocked {
		item.IsLocked = true
	}
	item.Save()
	w.Write([]byte("success"))
}

func AdTagArchiveHandler(w http.ResponseWriter, r *http.Request) {
	adTagID, adTagIDErr := getUintIDFromRequest(r, "ad_tag_id")
	if adTagIDErr != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.AdTag{}
	item.GetByID(adTagID, r.Context().Value("userID").(uint64))

	item.IsArchived = true
	item.IsActive = false

	item.Save()
	w.Write([]byte("success"))
}

func GetPublisherForAdTagHandler(w http.ResponseWriter, r *http.Request) {
	adTagID, adTagIDErr := getUintIDFromRequest(r, "ad_tag_id")
	if adTagIDErr != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	response := createAdTagPublishersList(adTagID)
	json_, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func AdTagsListJsonByAdvertiser(w http.ResponseWriter, r *http.Request) {
	advertiserID, err := getUintIDFromRequest(r, "advertiser_id")

	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getAdTagsListByAdvertiser(advertiserID)
	json_, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func AdTagsListJsonByPublisher(w http.ResponseWriter, r *http.Request) {
	publisherID, err := getUintIDFromRequest(r, "publisher_id")

	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getAdTagsListByPublisher(publisherID)
	json_, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func AdTagsListJson(w http.ResponseWriter, r *http.Request) {
	response := getAdTagsList()
	json_, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func AdTagsListJsonByAdvertAndPub(w http.ResponseWriter, r *http.Request) {
	advertiserID, errAdvert := getUintIDFromRequest(r, "advertiser_id")
	publisherID, errPub := getUintIDFromRequest(r, "publisher_id")

	if errAdvert != nil || errPub != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))

		return
	}

	response := getAdTagsListByAdvertiserAndPublisher(advertiserID, publisherID)
	json_, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_)
}

func createAdTagPublishersList(adTagID uint64) []publishersListForAdTag {
	items := []models.AdTagPublisher{}
	database.Postgres.Preload("Publisher").Where("ad_tag_id = ?", adTagID).Find(&items)
	var response []publishersListForAdTag
	response = make([]publishersListForAdTag, len(items))

	for i, item := range items {
		response[i] = publishersListForAdTag{
			AdTagPubID: item.ID,
			Name:       fmt.Sprintf(`%s - %s`, item.Publisher.Name, item.Name),
		}
	}
	return response
}

func getAdTagsListByAdvertiser(advertiserID uint64) []responseSelectList {
	items := []models.AdTag{}

	database.Postgres.
		Where("advertiser_id = ?", advertiserID).
		Where("is_active is true").
		Where("is_archived is not true").
		Order("name").
		Find(&items)

	var response []responseSelectList
	response = make([]responseSelectList, len(items))

	for i, item := range items {
		response[i] = responseSelectList{
			ID:   item.ID,
			Name: item.Name,
		}
	}

	return response
}

func getAdTagsList() []responseSelectList {
	var response []responseSelectList
	items := []models.AdTag{}

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

func getAdTagsListByPublisher(publisherID uint64) []responseSelectList {
	var response []responseSelectList
	items := []models.AdTag{}

	database.Postgres.Model(&models.AdTag{}).
		Joins("JOIN ad_tag_publishers ON ad_tags.id=ad_tag_publishers.ad_tag_id").
		Where("ad_tag_publishers.publisher_id = ?", publisherID).
		Order("name").
		Group("ad_tags.id").
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

func getAdTagsListByAdvertiserAndPublisher(advertiserID uint64, publisherID uint64) []responseSelectList {
	var response []responseSelectList
	items := []models.AdTag{}

	database.Postgres.Model(&models.AdTag{}).
		Joins("JOIN ad_tag_publishers ON ad_tags.id=ad_tag_publishers.ad_tag_id").
		Where("advertiser_id = ?", advertiserID).
		Where("ad_tag_publishers.publisher_id = ?", publisherID).
		Order("name").
		Group("ad_tags.id").
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

func FindDeadTagsHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.AdTag

	database.Postgres.
		Where("is_active = true AND (is_archived = false OR is_archived is null)").
		Where("created_at < (now() - interval '1 month')").
		Find(&list)

	idsList := make([]string, len(list))
	for i, adTag := range list {
		idsList[i] = strconv.Itoa(int(adTag.ID))
	}

	idsListJoined := strings.Join(idsList, ",")

	idsWithRequests := getAdTagIdsWithRequestsForLastMonth(idsListJoined)

	idsNeededToBeArchived := []string{}
	var hasAtLeastOneElementToArchive bool

	for _, id := range idsList {
		var found bool
		for _, idWithRequests := range idsWithRequests {
			if id == idWithRequests {
				found = true
				break
			}
		}
		if !found {
			idsNeededToBeArchived = append(idsNeededToBeArchived, id)
			hasAtLeastOneElementToArchive = true
		}
	}

	if hasAtLeastOneElementToArchive {
		database.Postgres.
			Table("ad_tags").
			Where("id IN (?)", idsNeededToBeArchived).
			Updates(map[string]interface{}{"is_archived": true, "is_active": false})
		log.Info(fmt.Sprintf("%d items were archived", len(idsNeededToBeArchived)))
	} else {
		log.Info("No items to archive")
	}
}

func getAdTagIdsWithRequestsForLastMonth(idsList string) []string {
	queryString := fmt.Sprintf(`
		SELECT
			distinct ad_tag_id
		FROM statistics.statistics_merged
		WHERE date >= today() - 30 AND ad_tag_id in (%s)
	`, idsList)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item string
	idsListWithRequests := []string{}
	for iter.Scan(
		&item,
	) {
		idsListWithRequests = append(idsListWithRequests, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return idsListWithRequests
}
