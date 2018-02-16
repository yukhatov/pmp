package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type AdvertiserPlatformType struct {
	ID   uint64 `gorm:"primary_key"`
	Name string `gorm:"size:255" valid:"required"`
}

func (platformtType *AdvertiserPlatformType) Save() bool {
	return len(database.Postgres.Save(platformtType).GetErrors()) == 0
}

func (platformtType *AdvertiserPlatformType) Create() bool {
	return len(database.Postgres.Create(platformtType).GetErrors()) == 0
}

func (platformtType *AdvertiserPlatformType) Delete() bool {
	return len(database.Postgres.Delete(platformtType).GetErrors()) == 0
}

func (platformtType *AdvertiserPlatformType) PopulateData(r *http.Request) {
	platformtType.Name = r.Form.Get("name")
}

func (platformtType *AdvertiserPlatformType) GetByID(id interface{}) {
	platformtType.ID = id.(uint64)
	database.Postgres.First(platformtType)
}
