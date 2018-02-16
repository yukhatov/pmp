package models

import (
	"time"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type Session struct {
	Key       string `gorm:"primary_key"`
	UserID    uint64
	LastLogin time.Time
	ExpireAt  time.Time

	User User
}

func (s *Session) Create(userID uint64) {
	s.Key = randStringBytes(50)
	s.UserID = userID
	s.LastLogin = time.Now()
	s.ExpireAt = time.Now().Add(7 * 24 * time.Hour)
	database.Postgres.Create(s)
}

func (s *Session) GetByID(id interface{}) {
	s.Key = id.(string)
	database.Postgres.Preload("User").First(s)
}

func (s *Session) IsExpired() bool {
	if time.Now().After(s.ExpireAt) {
		database.Postgres.Delete(s)
		return true
	}
	return false
}

func (s *Session) Save() {
	database.Postgres.Save(s)
}
