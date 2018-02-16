package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type PublisherLinkAdTagPublisher struct {
	ID               uint64 `gorm:"primary_key" json:"id"`
	PublisherLinkID  string `gorm:"size:8" valid:"required"`
	AdTagPublisherID string `gorm:"size:8" valid:"required"`
	IsActive         bool

	AdTagPublisher AdTagPublisher `gorm:"ForeignKey:ID;AssociationForeignKey:AdTagPublisherID;save_associations:false" valid:"-" json:"-"`
	PublisherLink  PublisherLink  `gorm:"ForeignKey:ID;AssociationForeignKey:PublisherLinkID;save_associations:false" valid:"-" json:"-"`
}

func (linkAdTag *PublisherLinkAdTagPublisher) Save() bool {
	return len(database.Postgres.Save(linkAdTag).GetErrors()) == 0
}

func (linkAdTag *PublisherLinkAdTagPublisher) Create() bool {
	return len(database.Postgres.Create(linkAdTag).GetErrors()) == 0
}

func (linkAdTag *PublisherLinkAdTagPublisher) PopulateData(r *http.Request) {
	linkAdTag.AdTagPublisherID = r.Form.Get("ad_tag_publisher_id")
}

func (linkAdTag *PublisherLinkAdTagPublisher) GetByID(id interface{}) {
	linkAdTag.ID = id.(uint64)
	database.Postgres.First(linkAdTag)
}
