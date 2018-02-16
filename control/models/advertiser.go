package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type Advertiser struct {
	ID                uint64 `gorm:"primary_key"`
	Name              string `gorm:"size:255" valid:"required"`
	CustomID          uint64 `valid:"required"`
	Comments          string
	RTBIntegrationUrl string
	IsDSP             bool
	InvoiceInfo       string

	AdTags []AdTag
}

func (ad *Advertiser) GetByID(id interface{}) {
	ad.ID = id.(uint64)
	database.Postgres.First(ad)
}

func (ad *Advertiser) Save() (bool, []string) {
	errors := database.Postgres.Save(ad).GetErrors()
	var messages []string

	if len(errors) > 0 {
		for _, error := range errors {
			messages = append(messages, getError(error).Detail)
		}
	}

	return len(errors) == 0, messages
}

func (ad *Advertiser) Create() bool {
	return len(database.Postgres.Create(ad).GetErrors()) == 0
}

func (ad *Advertiser) PopulateData(r *http.Request) {
	ad.Name = r.Form.Get("name")
	ad.Comments = r.Form.Get("comments")
	ad.IsDSP = r.Form.Get("is_dsp") == "enabled"
	ad.RTBIntegrationUrl = r.Form.Get("rtb_integration_url")
	ad.CustomID, _ = getUintValueFromForm(r, "custom_id", true)
	ad.InvoiceInfo = r.Form.Get("invoice_info")
}

func (ad *Advertiser) UpdateData(r *http.Request) {
	ad.Name = r.Form.Get("name")
	ad.Comments = r.Form.Get("comments")
	ad.IsDSP = r.Form.Get("is_dsp") == "enabled"
	ad.RTBIntegrationUrl = r.Form.Get("rtb_integration_url")
	ad.CustomID, _ = getUintValueFromForm(r, "custom_id", true)
	ad.InvoiceInfo = r.Form.Get("invoice_info")
}
