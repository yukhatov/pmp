package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"

	"encoding/json"

	"fmt"
	"time"

	"net/url"
	"strings"

	"bitbucket.org/tapgerine/pmp/control/config"
)

type AdTagPublisher struct {
	ID          string  `gorm:"primary_key" json:"id"`
	Name        string  `json:"name"`
	AdTagID     uint64  `valid:"required" json:"ad_tag_id"`
	PublisherID uint64  `valid:"required" json:"publisher_id"`
	Price       float64 `valid:"required,priceCompareValidator~Price should be lower than tag price,samePubValidator~Same publisher with same price exists,priceRangeValidator~Price should be between 1 and 50" json:"price"`
	URL         string  `json:"url"`
	IsActive    bool    `json:"is_active"`
	IsLocked    bool    `json:"is_locked"`
	//ShaveRequests    uint64  `valid:"shaveValidator~Shave should be between 0 and 99"`
	//ShaveImpressions uint64  `valid:"shaveValidator~Shave should be between 0 and 99"`

	Publisher                   Publisher                     `gorm:"ForeignKey:ID;AssociationForeignKey:PublisherID;save_associations:false" valid:"-" json:"-"`
	AdTag                       AdTag                         `gorm:"ForeignKey:ID;AssociationForeignKey:AdTagID;save_associations:false" valid:"-" json:"-"`
	PublisherLinkAdTagPublisher []PublisherLinkAdTagPublisher `gorm:"ForeignKey:AdTagPublisherID;AssociationForeignKey:ID;save_associations:false" valid:"-" json:"-"`

	CurrentUserID uint64 `gorm:"-" json:"-"`
}

func (atp *AdTagPublisher) GetByID(id interface{}, userID uint64) {
	atp.ID = id.(string)
	database.Postgres.Preload("Publisher").First(atp)
	atp.CurrentUserID = userID
}

func (atp *AdTagPublisher) Save() {
	database.Postgres.Save(atp)
}

func (atp *AdTagPublisher) MapURL(originalURL string, adUnitPubID string, domain string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(originalURL))

	if err != nil {
		return "", err
	}

	mappedURL := url.URL{
		Scheme: "https",
		Host:   domain,
		Path:   "rotator",
	}

	mappedURLQuery := mappedURL.Query()

	for param, value := range parsedURL.Query() {
		mappedURLQuery.Add(param, value[0])
	}
	mappedURLQuery.Add("adtagpubid", adUnitPubID)

	mappedURL.RawQuery, _ = url.QueryUnescape(mappedURLQuery.Encode())

	return mappedURL.String(), nil
}

func (atp *AdTagPublisher) Create() bool {
	atp.GenerateName()
	atp.GenerateID()
	//atp.URL, _ = generator.MapURL(atp.AdTag.URL, atp.ID)
	atp.URL, _ = atp.MapURL(atp.AdTag.URL, atp.ID, config.RotatorDomain)

	if len(database.Postgres.Create(atp).GetErrors()) == 0 {
		atp.Save()

		return true
	}

	return false
}

func (atp *AdTagPublisher) AfterCreate() {
	log, _ := json.Marshal(atp)

	al := &ActionLog{
		Log:    log,
		Model:  "ad_tag_publisher",
		Action: "create",
		UserID: atp.CurrentUserID,
	}

	al.Create()
}

func (atp *AdTagPublisher) AfterSave() {
	log, _ := json.Marshal(atp)

	al := &ActionLog{
		Log:    log,
		Model:  "ad_tag_publisher",
		Action: "edit",
		UserID: atp.CurrentUserID,
	}

	al.Create()
}

func (atp *AdTagPublisher) PopulateData(r *http.Request) {
	/*atp.ID = randStringBytes(8)*/
	atp.PublisherID, _ = getUintValueFromForm(r, "publisher", true)
	atp.Price, _ = getFloatValueFromForm(r, "price", true)
	//atp.ShaveRequests, _ = getUintValueFromForm(r, "shave_requests", true)
	//atp.ShaveImpressions, _ = getUintValueFromForm(r, "shave_impressions", true)
	atp.CurrentUserID = r.Context().Value("userID").(uint64)
}

func (atp *AdTagPublisher) UpdateData(r *http.Request) {
	price, err := getFloatValueFromForm(r, "price", true)

	if err == nil {
		atp.Price = price
	}

	atp.Name = r.Form.Get("name")
	//atp.ShaveRequests, _ = getUintValueFromForm(r, "shave_requests", true)
	//atp.ShaveImpressions, _ = getUintValueFromForm(r, "shave_impressions", true)
}

func (atp *AdTagPublisher) GenerateID() {
	atp.ID = randStringBytes(8)
}

func (atp *AdTagPublisher) GenerateName() {
	atp.Name = fmt.Sprintf(
		"%d_%s_%.2f_%s_%d_%s_%s",
		atp.Publisher.CustomID,
		atp.AdTag.PlayerInfo,
		atp.Price,
		atp.AdTag.GeoInfo,
		atp.AdTag.Advertiser.CustomID,
		time.Now().UTC().Format("02.01.2006"),
		atp.AdTag.PlatformInfo,
	)
}
