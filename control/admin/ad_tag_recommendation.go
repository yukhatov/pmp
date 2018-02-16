package admin

import (
	"fmt"
	"html/template"
	"net/http"

	"time"

	"strconv"
	"strings"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	log "github.com/Sirupsen/logrus"
	"github.com/roistat/go-clickhouse"
)

func AdTagRecommendationListHandler(w http.ResponseWriter, r *http.Request) {
	var adTagsRecommendation []models.AdTagRecommendation
	database.Postgres.Preload("AdTag").Where("fixed=false").Find(&adTagsRecommendation)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/ad_tag/recommendation.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	t.ExecuteTemplate(w, "main", adTagsRecommendation)
}

func AdTagRecommendationProcessorHandler(w http.ResponseWriter, r *http.Request) {
	var adTagsToCheck []models.AdTag
	database.Postgres.Where("is_active = true").Find(&adTagsToCheck)

	var adTagsIds []string
	adTagsIds = make([]string, len(adTagsToCheck))

	for i, item := range adTagsToCheck {
		adTagsIds[i] = strconv.Itoa(int(item.ID))
	}

	yesterdayDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	type AdTagRecommendationStatsResponse struct {
		AdTagID     int32
		Requests    int32
		Impressions int32
	}

	queryString := fmt.Sprintf(`
		SELECT
			ad_tag_id,
			requests,
			impressions
		FROM (
			SELECT
				ad_tag_id,
				sum(requests) as requests
			FROM statistics.statistics_merged
			WHERE toDate(date_time, 'UTC') = toDate('%s') AND ad_tag_id IN (%s)
			GROUP BY ad_tag_id
		)
		ANY LEFT JOIN (
			SELECT
				ad_tag_id,
				sum(events_count) as impressions
			FROM statistics.statistics_events_merged
			WHERE toDate(date_time, 'UTC') = toDate('%s') AND ad_tag_id IN (%s)
			GROUP BY ad_tag_id
		)
		USING (ad_tag_id)`,
		yesterdayDate, strings.Join(adTagsIds, ","), yesterdayDate, strings.Join(adTagsIds, ","),
	)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var adTag models.AdTag
	var item AdTagRecommendationStatsResponse
	for iter.Scan(
		&item.AdTagID, &item.Requests, &item.Impressions,
	) {
		if item.Requests > 0 && item.Impressions == 0 {
			var recommendationRecord models.AdTagRecommendation
			database.Postgres.Where("ad_tag_id = ?", item.AdTagID).Where("fixed = false").Find(&recommendationRecord)
			if recommendationRecord.AdTagID == 0 {
				database.Postgres.Where("id = ?", item.AdTagID).First(&adTag)
				recommendationRecord.AdTagID = uint64(item.AdTagID)
				recommendationRecord.DoNotShow = false
				recommendationRecord.Fixed = false

				recommendationRecord.Recommendation = fmt.Sprintf(
					`There are some requests but no impressions for %s. Ad tag created: %s`,
					yesterdayDate, adTag.CreatedAt.Format("2006-01-02"),
				)
				recommendationRecord.Create()
			}
		}
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
}

func AdTagRecommendationFixedHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.AdTagRecommendation{}
	item.GetByID(id)

	item.Fixed = true

	item.Save()
	w.Write([]byte("success"))
}
