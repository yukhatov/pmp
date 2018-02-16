package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type Parameter struct {
	ID           uint64 `gorm:"primary_key"`
	Name         string `gorm:"size:255" valid:"required"`
	Shortcut     string `gorm:"size:255" valid:"required"`
	Macros       string `gorm:"size:255" valid:"required"`
	DefaultValue string `gorm:"size:255"`
	Platform     string `valid:"required"`
}

func (param *Parameter) Save() bool {
	return len(database.Postgres.Save(param).GetErrors()) == 0
}

func (param *Parameter) Create() bool {
	return len(database.Postgres.Create(param).GetErrors()) == 0
}

func (param *Parameter) Delete() bool {
	return len(database.Postgres.Delete(param).GetErrors()) == 0
}

func (param *Parameter) PopulateData(r *http.Request) {
	param.Name = r.Form.Get("name")
	param.Shortcut = r.Form.Get("shortcut")
	param.Macros = r.Form.Get("macros")
	param.DefaultValue = r.Form.Get("default_value")
	param.Platform = r.Form.Get("platform")
}

func (param *Parameter) GetByID(id interface{}) {
	param.ID = id.(uint64)
	database.Postgres.First(param)
}
