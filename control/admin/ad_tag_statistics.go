package admin

import (
	"html/template"
	"net/http"

	"fmt"

	"strconv"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
)

type responseAdTagStatistics struct {
	AdTag                   models.AdTag
	ImpressionsPerPublisher map[string]int
}

func AdTagStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	adTag := &models.AdTag{}
	database.Postgres.Preload("AdTagPublishers").Preload("AdTagPublishers.Publisher").Find(adTag)

	var impressionsPerPublisher map[string]int
	impressionsPerPublisher = make(map[string]int, len(adTag.AdTagPublishers))

	for _, item := range adTag.AdTagPublishers {
		impressions, err := database.Redis.Get(fmt.Sprintf("request:%s", item.ID)).Result()
		if err != nil {
			continue
		}
		impressionsPerPublisher[item.ID], _ = strconv.Atoi(impressions)
	}

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/statistics.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseAdTagStatistics{AdTag: *adTag, ImpressionsPerPublisher: impressionsPerPublisher})
}
