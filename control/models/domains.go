package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"

	"strings"

	"errors"
	"net/url"

	"fmt"

	"github.com/lib/pq"
)

type DomainsList struct {
	ID      uint64         `gorm:"primary_key"`
	Name    string         `gorm:"size:255" valid:"required"`
	Domains pq.StringArray `gorm:"type:text[]" valid:"required"`
	Type    string
}

func (d *DomainsList) GetByID(id interface{}) {
	d.ID = id.(uint64)
	database.Postgres.First(d)
}

func (d *DomainsList) Save() {
	database.Postgres.Save(d)
}

func (d *DomainsList) Create() {
	database.Postgres.Create(d)
}

func (d *DomainsList) PopulateData(r *http.Request) {
	//var err error
	d.Name = r.Form.Get("name")
	d.Type = r.Form.Get("type")

	domainsSplit := strings.Split(r.Form.Get("domains"), "\r\n")

	var domainsParsed []string
	domainsParsed = make([]string, len(domainsSplit))
	var itemsAdded int
	for _, domain := range domainsSplit {
		parsed, err := getDomainNameFromURL(domain)
		if err == nil {
			domainsParsed[itemsAdded] = parsed
			itemsAdded++
		}
	}

	d.Domains = domainsParsed[:itemsAdded]
}

func (d *DomainsList) UpdateData(r *http.Request) {
	d.PopulateData(r)
}

func getDomainNameFromURL(URL string) (string, error) {
	var result string

	if URL != "" {
		URL = strings.ToLower(URL)
		URL = strings.Replace(URL, "www.", "", 1)
	} else {
		return result, errors.New("empty string given")
	}

	if !strings.HasPrefix(URL, "http") {
		URL = fmt.Sprintf("http://%s", URL)
	}

	parsed, err := url.Parse(URL)

	if err != nil {
		URL = strings.Replace(URL, "%", "", -1)
		parsed, err = url.Parse(URL)
		if err == nil {
			result = parsed.Hostname()
			err = nil
		}
	} else {
		result = parsed.Hostname()
	}

	if err == nil {
		split := strings.Split(result, ".")
		if len(split) < 2 {
			result = ""
			err = errors.New("wrong domain format")
		}
	}

	return result, err
}
