package admin

import (
	"fmt"
	"html/template"
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"

	"strings"

	"github.com/asaskevich/govalidator"
)

type responsePublisherList struct {
	Publishers []*models.AdTagPublisher
	AdTag      *models.AdTag
}

type responseAdTagPublisherEdit struct {
	Item       models.AdTagPublisher
	AdTag      models.AdTag
	Publishers []*models.Publisher
	IsEditing  bool
	Success    bool
	Errors     []string
}

func AdTagPublisherListHandler(w http.ResponseWriter, r *http.Request) {
	adTag := &models.AdTag{}
	adTag.GetByIDFromRequest(r)

	var adTagPublisherList []*models.AdTagPublisher
	database.Postgres.Preload("Publisher").Where("ad_tag_id = ?", adTag.ID).Find(&adTagPublisherList)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/publisher_list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responsePublisherList{Publishers: adTagPublisherList, AdTag: adTag})
}

func AdTagPublisherAddHandler(w http.ResponseWriter, r *http.Request) {
	adTag := &models.AdTag{}
	adTag.GetByIDFromRequest(r)

	var adTagPublisher models.AdTagPublisher
	var publisherList []*models.Publisher

	database.Postgres.Find(&publisherList)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/publisher_edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseAdTagPublisherEdit{
		Item:       adTagPublisher,
		AdTag:      *adTag,
		Publishers: publisherList,
		Errors:     strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func AdTagPublisherEditHandler(w http.ResponseWriter, r *http.Request) {
	adTag := &models.AdTag{}
	adTag.GetByIDFromRequest(r)

	var err error
	var success bool
	var publisherList []*models.Publisher
	database.Postgres.Find(&publisherList)

	adTagPublisherID, _ := getStringFromRequest(r, "id")
	isNewRecord := adTagPublisherID == "0"

	item := &models.AdTagPublisher{}

	if r.Method == "POST" {
		r.ParseForm()

		if isNewRecord {
			item.PopulateData(r)
			item.AdTagID = adTag.ID
			item.AdTag = *adTag

			publisher := models.Publisher{}
			publisher.GetByID(item.PublisherID)
			item.Publisher = publisher

		} else {
			item.GetByID(adTagPublisherID, r.Context().Value("userID").(uint64))
			item.UpdateData(r)
		}

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
		// TODO: move validation to model
		if _, err := govalidator.ValidateStruct(item); err != nil {
			errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
			if isNewRecord {
				http.Redirect(w, r, fmt.Sprintf("/ad_tag/%d/publisher/add/?error=%s", adTag.ID, errorReplaced), 302)
			} else {
				http.Redirect(w, r, fmt.Sprintf("/ad_tag/%d/publisher/edit/%s?error=%s", adTag.ID, item.ID, errorReplaced), 302)
			}
			return
		} else {
			if isNewRecord {
				/*item.GenerateName()*/
				item.Create()

			} else {
				item.Save()
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/ad_tag/%d/publisher/edit/%s?success=true", adTag.ID, item.ID), 302)
		return

	} else {
		if isNewRecord {
			// TODO: error handling
			panic("need ad unit publisher id")
		}

		success = len(r.URL.Query().Get("success")) > 0

		item.GetByID(adTagPublisherID, r.Context().Value("userID").(uint64))
	}

	_ = err

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/publisher_edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", responseAdTagPublisherEdit{
		Item:       *item,
		AdTag:      *adTag,
		Publishers: publisherList,
		Success:    success,
		Errors:     strings.Split(r.URL.Query().Get("error"), "|"),
		IsEditing:  true,
	})
}

func AdTagPublisherActivationHandler(w http.ResponseWriter, r *http.Request) {
	adTagPublisherID, err := getStringFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.AdTagPublisher{}
	item.GetByID(adTagPublisherID, r.Context().Value("userID").(uint64))

	item.IsActive = !item.IsActive

	if item.IsActive && !item.IsLocked {
		item.IsLocked = true
	}
	item.Save()
	w.Write([]byte("success"))
}
