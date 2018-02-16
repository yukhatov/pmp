package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type PublisherLink struct {
	ID            string `gorm:"primary_key"`
	Name          string `gorm:"size:255" valid:"required"`
	PublisherID   uint64 `valid:"required" json:"publisher_id"`
	Link          string
	DomainsListID uint64
	Platform      string
	Price         float64 `valid:"required,priceValidator~Price should be greater than 0" json:"price"`
	Optimization  string
	StudyRequests uint64

	PublisherLinkAdTagPublisher []PublisherLinkAdTagPublisher `gorm:"ForeignKey:PublisherLinkID;AssociationForeignKey:ID;save_associations:false" valid:"-" json:"-"`

	DomainsList DomainsList
	Publisher   Publisher

	HasAdTag string `gorm:"-"`
}

func (link *PublisherLink) GenerateID() {
	link.ID = randStringBytes(8)
}

func (link *PublisherLink) GetByID(id interface{}) {
	link.ID = id.(string)
	database.Postgres.Preload("PublisherLinkAdTagPublisher").Preload("PublisherLinkAdTagPublisher.AdTagPublisher").Preload("PublisherLinkAdTagPublisher.AdTagPublisher.AdTag").Preload("PublisherLinkAdTagPublisher.PublisherLink").First(link)
}

func (link *PublisherLink) Save() bool {
	return len(database.Postgres.Save(link).GetErrors()) == 0
}

func (link *PublisherLink) Create() bool {
	return len(database.Postgres.Create(link).GetErrors()) == 0
}

func (link *PublisherLink) PopulateData(r *http.Request) {
	link.ID = r.Form.Get("id")
	link.Price, _ = getFloatValueFromForm(r, "price", true)
	link.Name = r.Form.Get("name")
	link.Link = r.Form.Get("link")
	link.Platform = r.Form.Get("platform")
	link.DomainsListID, _ = getUintValueFromForm(r, "domains_list", false)
	link.Optimization = r.Form.Get("optimization")
	link.StudyRequests, _ = getUintValueFromForm(r, "study", true)
}

func (link *PublisherLink) UpdateData(r *http.Request) {
	link.Name = r.Form.Get("name")
	link.Link = r.Form.Get("link")
	link.Price, _ = getFloatValueFromForm(r, "price", true)
	link.DomainsListID, _ = getUintValueFromForm(r, "domains_list", false)
	link.Optimization = r.Form.Get("optimization")
	link.StudyRequests, _ = getUintValueFromForm(r, "study", true)
}
