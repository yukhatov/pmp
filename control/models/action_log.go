package models

import (
	"time"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type ActionLog struct {
	ID        uint64
	CreatedAt time.Time

	UserID uint64
	Model  string
	Action string
	Log    []byte

	User User
}

func (al *ActionLog) Create() {
	database.Postgres.Create(al)
}
