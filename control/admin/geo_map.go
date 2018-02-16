package admin

import (
	"fmt"
	"html/template"
	"net/http"

	"encoding/json"

	"bitbucket.org/tapgerine/pmp/control/database"
	log "github.com/Sirupsen/logrus"
	"github.com/roistat/go-clickhouse"
)

type clickHouseGeoResponse struct {
	Requests   int64  `json:"requests"`
	GeoCountry string `json:"geo_country"`
}

func GeoMapIndex(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/geo_map/list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", nil)
}

func GeoMapData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := getGeoStats()
	json_, _ := json.Marshal(stats)
	w.Write(json_)
}

func getGeoStats() []clickHouseGeoResponse {
	queryString := fmt.Sprintf(`
		SELECT
			sum(requests) as requests,
			geo_country
		FROM statistics.statistics_merged
		WHERE geo_country != ''
		GROUP BY geo_country`)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseGeoResponse
	var stats []clickHouseGeoResponse
	for iter.Scan(
		&item.Requests, &item.GeoCountry,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}
