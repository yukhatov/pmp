package models

import (
	"net/http"
	"time"

	"strings"

	"bitbucket.org/tapgerine/pmp/control/database"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`

	UserName        string
	FullName        string
	Password        []byte
	Email           string
	Skype           string
	Role            string
	PublisherID     uint64
	DefaultTimezone string
}

func (u *User) GetByID(id interface{}) {
	u.ID = id.(uint64)
	database.Postgres.First(u)
}

func (u *User) Save() {
	database.Postgres.Save(u)
}

func (u *User) Create() {
	database.Postgres.Create(u)
}

func (u *User) PopulateData(r *http.Request) {
	u.UserName = r.Form.Get("user_name")
	u.FullName = r.Form.Get("full_name")
	u.Email = r.Form.Get("email")
	u.DefaultTimezone = r.Form.Get("default_timezone")
	u.Password, _ = bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(r.Form.Get("password"))), bcrypt.DefaultCost)
}

func (u *User) UpdateData(r *http.Request) {
	u.UserName = r.Form.Get("user_name")
	u.FullName = r.Form.Get("full_name")
	u.Email = r.Form.Get("email")
	u.DefaultTimezone = r.Form.Get("default_timezone")
	password := strings.TrimSpace(r.Form.Get("password"))
	if password != "" {
		u.Password, _ = bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(r.Form.Get("password"))), bcrypt.DefaultCost)
	}
}
