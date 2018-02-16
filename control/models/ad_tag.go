package models

import (
	"net/http"

	"time"

	"strings"

	"encoding/json"

	"strconv"

	"bitbucket.org/tapgerine/pmp/control/database"
	"github.com/lib/pq"
)

type AdTag struct {
	ID        uint64 `gorm:"primary_key" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name                     string         `gorm:"size:255" valid:"required" json:"name"`
	AdvertiserID             uint64         `valid:"required" json:"advertiser_id"`
	Price                    float64        `valid:"required,priceValidator~Price should be greater than 0" json:"price"`
	MinimumMargin            float64        `valid:"required,marginValidator~Margin should be less than tag price" json:"minimum_margin"`
	IsActive                 bool           `json:"is_active"`
	IsLocked                 bool           `json:"is_locked"`
	URL                      string         `valid:"required,url" json:"url"`
	IsVast                   bool           `json:"is_vast"`
	GeoInfo                  string         `json:"geo_info"`
	PlayerInfo               string         `json:"player_info"`
	PlatformInfo             string         `json:"platform_info"`
	GeoCountry               pq.StringArray `json:"geo_country" gorm:"type:text[]"`
	OsName                   string
	BrowserName              string
	DeviceType               string
	IsTargeted               bool
	AdvertiserPlatformTypeID uint64
	IsArchived               bool
	DomainsListID            uint64

	Advertiser             Advertiser              `valid:"-" json:"-"`
	AdTagPublishers        []*AdTagPublisher       `valid:"-" json:"-"`
	AdvertiserPlatformType AdvertiserPlatformTypes `gorm:"ForeignKey:AdvertiserPlatformTypeID;AssociationForeignKey:ID;save_associations:false" valid:"-" json:"-"`
	DomainsList            DomainsList

	CurrentUserID uint64 `gorm:"-" json:"-"`
}

func (ad *AdTag) GetByID(id interface{}, userID uint64) {
	ad.ID = id.(uint64)
	database.Postgres.Preload("Advertiser").Preload("AdvertiserPlatformType").First(ad)
	ad.CurrentUserID = userID
}

func (ad *AdTag) GetByIDFromRequest(r *http.Request) {
	adTagID, adTagIDErr := getUintIDFromRequest(r, "ad_tag_id")

	if adTagIDErr != nil {
		return
	}
	ad.GetByID(adTagID, r.Context().Value("userID").(uint64))
}

func (ad *AdTag) GetByIDFromForm(r *http.Request, name string) {
	value := r.Form.Get(name)

	adTagID, adTagIDErr := strconv.ParseUint(value, 10, 0)

	if adTagIDErr != nil {
		return
	}

	ad.GetByID(adTagID, r.Context().Value("userID").(uint64))
}

func (ad *AdTag) Save() {
	database.Postgres.Save(ad)
}

func (ad *AdTag) Create() {
	database.Postgres.Create(ad)
}

func (ad *AdTag) AfterCreate() {
	log, _ := json.Marshal(ad)

	al := &ActionLog{
		Log:    log,
		Model:  "ad_tag",
		Action: "create",
		UserID: ad.CurrentUserID,
	}
	al.Create()
}

func (ad *AdTag) AfterSave() {
	log, _ := json.Marshal(ad)

	al := &ActionLog{
		Log:    log,
		Model:  "ad_tag",
		Action: "edit",
		UserID: ad.CurrentUserID,
	}
	al.Create()
}

func (ad *AdTag) PopulateData(r *http.Request) {
	//var err error
	ad.AdvertiserID, _ = getUintValueFromForm(r, "advertiser_id", true)
	ad.Name = r.Form.Get("name")
	ad.URL = strings.TrimSpace(r.Form.Get("url"))
	ad.Price, _ = getFloatValueFromForm(r, "price", true)
	ad.MinimumMargin, _ = getFloatValueFromForm(r, "minimum_margin", true)
	ad.IsVast = r.Form.Get("is_vast") == "enabled"
	ad.GeoInfo = strings.TrimSpace(r.Form.Get("geo_info"))
	ad.PlayerInfo = r.Form.Get("player_info")
	ad.PlatformInfo = r.Form.Get("platform_info")
	ad.GeoCountry = r.Form["geo_country"]
	ad.DeviceType = r.Form.Get("device_type")
	ad.IsTargeted = r.Form.Get("is_targeted") == "enabled"
	ad.AdvertiserPlatformTypeID, _ = getUintValueFromForm(r, "type_id", true)
	ad.DomainsListID, _ = getUintValueFromForm(r, "domains_list", false)
}

func (ad *AdTag) UpdateData(r *http.Request) {
	ad.Name = r.Form.Get("name")
	ad.URL = r.Form.Get("url")
	ad.Price, _ = getFloatValueFromForm(r, "price", true)
	ad.MinimumMargin, _ = getFloatValueFromForm(r, "minimum_margin", true)
	ad.IsVast = r.Form.Get("is_vast") == "enabled"
	ad.GeoInfo = strings.TrimSpace(r.Form.Get("geo_info"))
	ad.PlayerInfo = r.Form.Get("player_info")
	ad.PlatformInfo = r.Form.Get("platform_info")
	ad.GeoCountry = r.Form["geo_country"]
	ad.DeviceType = r.Form.Get("device_type")
	ad.IsTargeted = r.Form.Get("is_targeted") == "enabled"
	ad.AdvertiserPlatformTypeID, _ = getUintValueFromForm(r, "type_id", true)
	ad.DomainsListID, _ = getUintValueFromForm(r, "domains_list", false)
}
