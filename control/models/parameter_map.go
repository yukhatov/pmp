package models

import (
	"net/http"
	"strconv"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type ParametersMaps struct {
	ID                   uint64
	ParameterID          uint64
	AdvertiserPlatformID uint64
	Name                 string `gorm:"size:255" valid:"required"`
	Shortcut             string `gorm:"size:255" valid:"required"`
	Macros               string `gorm:"size:255" valid:"required"`
	DefaultValue         string `gorm:"size:255"`
	IsRequired           bool

	OriginalParameter      Parameter              `gorm:"ForeignKey:ParameterID;AssociationForeignKey:ID;save_associations:false" valid:"-" json:"-"`
	AdvertiserPlatformType AdvertiserPlatformType `gorm:"ForeignKey:AdvertiserPlatformID;AssociationForeignKey:ID;save_associations:false" valid:"-" json:"-"`
}

func (param *ParametersMaps) GetByID(id interface{}) {
	param.ID = id.(uint64)
	database.Postgres.First(param)
}

func (param *ParametersMaps) Save() bool {
	return len(database.Postgres.Save(param).GetErrors()) == 0
}

func (param *ParametersMaps) Create() bool {
	return len(database.Postgres.Create(param).GetErrors()) == 0
}

func (parameterMap *ParametersMaps) PopulateData(r *http.Request) {
	parameterMap.ParameterID, _ = strconv.ParseUint(r.Form.Get("parameter_id"), 10, 0)
	parameterMap.Name = r.Form.Get("name")
	parameterMap.Shortcut = r.Form.Get("shortcut")
	parameterMap.Macros = r.Form.Get("macros")
	parameterMap.DefaultValue = r.Form.Get("default_value")
	parameterMap.IsRequired = r.Form.Get("is_required") == "enabled"

	platformType := &AdvertiserPlatformType{}
	platformTypeID, _ := strconv.ParseUint(r.Form.Get("platform_type_id"), 10, 0)
	platformType.GetByID(platformTypeID)

	parameterMap.AdvertiserPlatformID = platformType.ID
}
