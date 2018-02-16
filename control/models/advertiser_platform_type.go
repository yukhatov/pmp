package models

import "bitbucket.org/tapgerine/pmp/control/database"

type AdvertiserPlatformTypes struct {
	ID   uint64 `gorm:"primary_key" json:"id"`
	Name string `gorm:"size:255" valid:"required" json:"name"`

	ParametersMaps []ParametersMaps `gorm:"ForeignKey:AdvertiserPlatformID;AssociationForeignKey:ID;save_associations:false" valid:"-" json:"-"`
}

func (param *AdvertiserPlatformTypes) GetByID(id interface{}) {
	param.ID = id.(uint64)
	database.Postgres.First(param)
}
