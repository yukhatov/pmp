package models

import (
	"net/http"
	"time"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type PublisherInvoice struct {
	ID            uint64 `gorm:"primary_key"`
	PublisherID   uint64 `valid:"required" json:"publisher_id"`
	DateCreated   time.Time
	DatePaid      time.Time
	DueDate       time.Time
	Status        string
	FileName      string
	FilePath      string
	Amount        float64
	Fee           float64
	PayTerms      string
	Notes         string
	InvoiceNumber uint64

	Publisher Publisher `gorm:"ForeignKey:ID;AssociationForeignKey:PublisherID;save_associations:false" valid:"-" json:"-"`
}

func (pub *PublisherInvoice) GetByID(id interface{}) {
	pub.ID = id.(uint64)
	database.Postgres.Preload("Publisher").First(pub)
}

func (pub *PublisherInvoice) Save() {
	database.Postgres.Save(pub)
}

func (pub *PublisherInvoice) Create() {
	database.Postgres.Create(pub)
}

func (pub *PublisherInvoice) PopulateData(r *http.Request) {
	pub.PublisherID, _ = getUintValueFromForm(r, "publisher", true)
	pub.Amount, _ = getFloatValueFromForm(r, "amount", true)
	pub.Fee, _ = getFloatValueFromForm(r, "fee", true)
	pub.PayTerms = r.Form.Get("pay_terms")
	pub.Notes = r.Form.Get("notes")
	pub.DateCreated = time.Now()
	pub.Status = "Unpaid"
	pub.InvoiceNumber, _ = getUintValueFromForm(r, "invoice_number", false)
}

func (pub *PublisherInvoice) UpdateData(r *http.Request) {
	pub.Amount, _ = getFloatValueFromForm(r, "amount", true)
	pub.Fee, _ = getFloatValueFromForm(r, "fee", true)
	pub.PayTerms = r.Form.Get("pay_terms")
	pub.Notes = r.Form.Get("notes")
	pub.InvoiceNumber, _ = getUintValueFromForm(r, "invoice_number", false)
}
