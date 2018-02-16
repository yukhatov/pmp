package models

import (
	"net/http"
	"time"

	"bitbucket.org/tapgerine/pmp/control/database"
)

const INVOICE_START_NUMBER uint64 = 1812010085

type AdvertiserInvoice struct {
	ID            uint64 `gorm:"primary_key"`
	AdvertiserID  uint64 `valid:"required" json:"advertiser_id"`
	InvoiceNumber uint64
	DateCreated   time.Time
	DateFrom      string
	DatePaid      time.Time
	DueDate       string
	Description   string
	Status        string
	//FileName     string
	//FilePath     string
	Amount float64

	Advertiser Advertiser `valid:"-" json:"-"`
}

func (adv *AdvertiserInvoice) GetByID(id interface{}) {
	adv.ID = id.(uint64)
	database.Postgres.Preload("Advertiser").First(adv)
}

func (adv *AdvertiserInvoice) Save() {
	database.Postgres.Save(adv)
}

func (adv *AdvertiserInvoice) Create() {
	database.Postgres.Create(adv)
	//adv.InvoiceNumber = INVOICE_START_NUMBER + adv.ID
	adv.Save()
}

func (adv *AdvertiserInvoice) PopulateData(r *http.Request) {
	adv.AdvertiserID, _ = getUintValueFromForm(r, "advertiser", true)
	adv.Amount, _ = getFloatValueFromForm(r, "amount", true)
	adv.DateFrom = r.Form.Get("date_from")
	adv.DueDate = r.Form.Get("due_date")
	adv.Description = r.Form.Get("description")
	adv.DateCreated = time.Now()
	adv.Status = "Unpaid"
	adv.InvoiceNumber, _ = getUintValueFromForm(r, "number", true)
}

func (adv *AdvertiserInvoice) UpdateData(r *http.Request) {
	adv.Amount, _ = getFloatValueFromForm(r, "amount", true)
	adv.DateFrom = r.Form.Get("date_from")
	adv.DueDate = r.Form.Get("due_date")
	adv.Description = r.Form.Get("description")
	adv.InvoiceNumber, _ = getUintValueFromForm(r, "number", true)
}
