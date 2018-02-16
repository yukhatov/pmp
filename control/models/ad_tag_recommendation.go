package models

import "bitbucket.org/tapgerine/pmp/control/database"

type AdTagRecommendation struct {
	ID uint64 `gorm:"primary_key" json:"id"`

	AdTagID        uint64
	Recommendation string
	DoNotShow      bool
	Fixed          bool

	AdTag AdTag `gorm:"ForeignKey:ID;AssociationForeignKey:AdTagID;save_associations:false" valid:"-" json:"-"`
}

func (ad *AdTagRecommendation) GetByID(id interface{}) {
	ad.ID = id.(uint64)
	database.Postgres.Preload("AdTag").First(ad)
}

func (ad *AdTagRecommendation) Save() {
	database.Postgres.Save(ad)
}

func (ad *AdTagRecommendation) Create() {
	database.Postgres.Create(ad)
}
