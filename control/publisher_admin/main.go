package publisherAdmin

import (
	"html/template"
	"net/http"

	"fmt"

	"time"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	log "github.com/Sirupsen/logrus"
	"github.com/roistat/go-clickhouse"
)

type LeftSidebar struct {
	ActiveTags  int
	AdRequests  int64
	Impressions int64
	Income      float64
}

type responseMainPage struct {
	User        models.User
	LeftSidebar LeftSidebar
	Stats       map[string][]clickHouseResponse
	TotalStats  map[string]totalStats
	Search      string
	StartDate   string
	EndDate     string
}

type clickHouseResponse struct {
	PublisherAdTagID string
	PublisherTagName string
	AdTagPublisherID string
	Requests         int64
	Date             string
	DateTime         string
	Impressions      int64
	Amount           float64
	FillRate         float64
}

type totalStats struct {
	Requests    int64
	Impressions int64
	Amount      float64
}

// TODO: remove, it's deprecated
type AdTagShave struct {
	Requests    uint64
	Impressions uint64
}

func calculateFillRate(impressions, requests int64) string {
	return fmt.Sprintf("%.2f", float64(impressions)/float64(requests)*100.0)
}

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	sessionCookie, _ := r.Cookie("Session")
	session := &models.Session{}
	session.GetByID(sessionCookie.Value)

	var startDate, endDate, search string
	if len(r.URL.Query()) > 0 {
		startDate = r.URL.Query().Get("start_date")
		endDate = r.URL.Query().Get("end_date")
		search = r.URL.Query().Get("search")
	}

	if startDate == "" {
		startDate = time.Now().Add(7 * -24 * time.Hour).UTC().Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().UTC().Format("2006-01-02")
	}

	fm := template.FuncMap{"calculateFillRate": calculateFillRate}

	t, _ := template.New("main").Funcs(fm).ParseFiles(
		"control/templates/publisher_admin/main.html",
	)

	statsSorted, statsTotalByTag := collectStatistics(session.User.PublisherID, search, session.User.DefaultTimezone, startDate, endDate)

	statsSortedTargeting, statsTotalByTagTargeting := collectStatisticsForPublisherLinks(session.User.PublisherID, search, session.User.DefaultTimezone, startDate, endDate)

	for k, v := range statsSortedTargeting {
		statsSorted[k] = v
	}

	for k, v := range statsTotalByTagTargeting {
		statsTotalByTag[k] = v
	}

	t.ExecuteTemplate(w, "main", responseMainPage{
		User:        session.User,
		LeftSidebar: collectDataForLeftSidebar(session.User.PublisherID, session.User.DefaultTimezone),
		Stats:       statsSorted,
		TotalStats:  statsTotalByTag,
		Search:      search,
		StartDate:   startDate,
		EndDate:     endDate,
	})
}

func collectStatistics(publisherID uint64, search string, timezone, startDate, endDate string) (map[string][]clickHouseResponse, map[string]totalStats) {
	query := clickhouse.NewQuery(`
		SELECT
			dictGetStringOrDefault('ad_tag_publisher', 'name', tuple(ad_tag_publisher_id), 'VPAID component') as publisher_tag_name,
			ad_tag_publisher_id,
			requests,
			impressions,
			amount,
			date_with_timezone
		FROM (
			SELECT
				ad_tag_publisher_id,
				sum(requests) as requests,
				toDate(date_time, ?) as date_with_timezone
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate(?) AND date_with_timezone <= toDate(?) AND publisher_id = toUInt32(?) AND request_type = 'direct'
			GROUP BY ad_tag_publisher_id, date_with_timezone
		)
		ANY LEFT JOIN (
			SELECT
				ad_tag_publisher_id,
				sum(events_count) as impressions,
				sum(amount) / 1000000 as amount,
				toDate(date_time, ?) as date_with_timezone
			FROM statistics.statistics_events_merged
			WHERE date_with_timezone >= toDate(?) AND date_with_timezone <= toDate(?) AND publisher_id = toUInt32(?) AND request_type = 'direct' AND event_name = 'impression'
			GROUP BY ad_tag_publisher_id, date_with_timezone
		) USING (ad_tag_publisher_id, date_with_timezone)
		WHERE publisher_tag_name LIKE ? AND publisher_tag_name != 'some name'
		ORDER BY date_with_timezone`,
		timezone, startDate, endDate, publisherID, timezone, startDate, endDate, publisherID, fmt.Sprintf(`%%%s%%`, search))

	iter := query.Iter(database.ClickHouse)

	statsSorted := make(map[string][]clickHouseResponse)
	statsTotalByTag := make(map[string]totalStats)

	publisherAdTagsShave := getPublisherAdTagsShave(publisherID)

	var item clickHouseResponse
	for iter.Scan(
		&item.PublisherTagName, &item.PublisherAdTagID, &item.Requests, &item.Impressions,
		&item.Amount, &item.Date,
	) {

		shave := publisherAdTagsShave[item.PublisherAdTagID]
		if shave.Requests > 0 {
			item.Requests = calculateRequestsShave(shave.Requests, item.Requests)
			//item.Requests = int32(float64(item.Requests) * (1.0 - float64(shave.Requests)/100.0))
		}

		//if shave.Impressions > 0 && item.Impressions > 1 {
		if shave.Impressions > 0 {
			item.Impressions, item.Amount = calculateImpressionsShave(shave.Impressions, item.Impressions, item.Amount)
			//defaultAmount := (item.Amount / float64(item.Impressions)) * float64(1000)
			//item.Impressions = int32(float64(item.Impressions) * (1.0 - float64(shave.Impressions)/100.0))
			//item.Amount = (float64(item.Impressions) * defaultAmount) / float64(1000)
		}

		item.FillRate = float64(item.Impressions) / float64(item.Requests) * 100.0
		statsSorted[item.PublisherTagName] = append(statsSorted[item.PublisherTagName], item)

		if _, ok := statsTotalByTag[item.PublisherTagName]; !ok {
			statsTotalByTag[item.PublisherTagName] = totalStats{
				Requests:    item.Requests,
				Impressions: item.Impressions,
				Amount:      item.Amount,
			}
		} else {
			total := statsTotalByTag[item.PublisherTagName]
			total.Requests += item.Requests
			total.Impressions += item.Impressions
			total.Amount += item.Amount
			statsTotalByTag[item.PublisherTagName] = total
		}

	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return statsSorted, statsTotalByTag
}

func collectStatisticsForPublisherLinks(publisherID uint64, search string, timezone, startDate, endDate string) (map[string][]clickHouseResponse, map[string]totalStats) {
	query := clickhouse.NewQuery(`
		SELECT
			dictGetStringOrDefault('publisher_links', 'name', tuple(targeting_id), 'Tapgerine tag') as publisher_link_name,
			targeting_id,
			requests,
			impressions,
			amount,
			date_with_timezone
		FROM (
			SELECT
				targeting_id,
				sum(requests) as requests,
				toDate(date_time, ?) as date_with_timezone
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate(?) AND date_with_timezone <= toDate(?) AND publisher_id = toUInt32(?) AND request_type != 'direct'
			GROUP BY targeting_id, date_with_timezone
		)
		ANY LEFT JOIN (
			SELECT
				targeting_id,
				sum(events_count) as impressions,
				sum(amount) / 1000000 as amount,
				toDate(date_time, ?) as date_with_timezone
			FROM statistics.statistics_events_merged
			WHERE date_with_timezone >= toDate(?) AND date_with_timezone <= toDate(?) AND publisher_id = toUInt32(?) AND request_type != 'direct' AND event_name = 'impression'
			GROUP BY targeting_id, date_with_timezone
		) USING (targeting_id, date_with_timezone)
		WHERE publisher_link_name LIKE ? AND publisher_link_name != 'some name'
		ORDER BY date_with_timezone`,
		timezone, startDate, endDate, publisherID, timezone, startDate, endDate, publisherID, fmt.Sprintf(`%%%s%%`, search))

	iter := query.Iter(database.ClickHouse)

	statsSorted := make(map[string][]clickHouseResponse)
	statsTotalByTag := make(map[string]totalStats)

	//publisherAdTagsShave := getPublisherAdTagsShave(publisherID)

	var item clickHouseResponse
	for iter.Scan(
		&item.PublisherTagName, &item.PublisherAdTagID, &item.Requests, &item.Impressions,
		&item.Amount, &item.Date,
	) {

		//shave := publisherAdTagsShave[item.PublisherAdTagID]
		//if shave.Requests > 0 {
		//	item.Requests = calculateRequestsShave(shave.Requests, item.Requests)
		//	//item.Requests = int32(float64(item.Requests) * (1.0 - float64(shave.Requests)/100.0))
		//}

		//if shave.Impressions > 0 && item.Impressions > 1 {
		//if shave.Impressions > 0 {
		//	item.Impressions, item.Amount = calculateImpressionsShave(shave.Impressions, item.Impressions, item.Amount)
		//	//defaultAmount := (item.Amount / float64(item.Impressions)) * float64(1000)
		//	//item.Impressions = int32(float64(item.Impressions) * (1.0 - float64(shave.Impressions)/100.0))
		//	//item.Amount = (float64(item.Impressions) * defaultAmount) / float64(1000)
		//}

		item.FillRate = float64(item.Impressions) / float64(item.Requests) * 100.0
		statsSorted[item.PublisherTagName] = append(statsSorted[item.PublisherTagName], item)

		if _, ok := statsTotalByTag[item.PublisherTagName]; !ok {
			statsTotalByTag[item.PublisherTagName] = totalStats{
				Requests:    item.Requests,
				Impressions: item.Impressions,
				Amount:      item.Amount,
			}
		} else {
			total := statsTotalByTag[item.PublisherTagName]
			total.Requests += item.Requests
			total.Impressions += item.Impressions
			total.Amount += item.Amount
			statsTotalByTag[item.PublisherTagName] = total
		}

	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return statsSorted, statsTotalByTag
}

func collectDataForLeftSidebar(publisherID uint64, timezone string) LeftSidebar {
	sidebar := LeftSidebar{}

	database.Postgres.
		Model(&models.AdTagPublisher{}).
		Where(&models.AdTagPublisher{
			IsActive:    true,
			PublisherID: publisherID,
		}).
		Count(&sidebar.ActiveTags)

	todayDate := time.Now().UTC().Format("2006-01-02")

	query := clickhouse.NewQuery(`
		SELECT
			requests,
			impressions,
			ad_tag_publisher_id,
			toFloat64(amount)
		FROM (
			SELECT
				ad_tag_publisher_id,
				sum(requests) as requests
			FROM statistics.statistics_merged
			WHERE toDate(date_time, ?) = toDate(?) AND publisher_id = toUInt32(?)
			GROUP BY ad_tag_publisher_id
		)
		ANY LEFT JOIN (
			SELECT
				ad_tag_publisher_id,
				sum(events_count) as impressions,
				sum(amount) / 1000000 as amount
			FROM statistics.statistics_events_merged
			WHERE toDate(date_time, ?) == toDate(?) AND publisher_id = toUInt32(?)
			GROUP BY ad_tag_publisher_id
		) USING (ad_tag_publisher_id)`,
		timezone, todayDate, publisherID, timezone, todayDate, publisherID)

	publisherAdTagsShave := getPublisherAdTagsShave(publisherID)

	type queryResponse struct {
		Requests         int64
		Impressions      int64
		PublisherAdTagID string
		Amount           float64
	}

	var item queryResponse
	iter := query.Iter(database.ClickHouse)
	for iter.Scan(&item.Requests, &item.Impressions, &item.PublisherAdTagID, &item.Amount) {
		shave := publisherAdTagsShave[item.PublisherAdTagID]

		if shave.Requests > 0 {
			item.Requests = calculateRequestsShave(shave.Requests, item.Requests)
		}
		if shave.Impressions > 0 {
			item.Impressions, item.Amount = calculateImpressionsShave(shave.Impressions, item.Impressions, item.Amount)
		}

		sidebar.AdRequests += item.Requests
		sidebar.Impressions += item.Impressions
		sidebar.Income += item.Amount
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return sidebar
}

func getPublisherAdTagsShave(publisherID uint64) map[string]AdTagShave {
	// Collecting all publisher tags to perform shave
	var publisherAdTags []models.AdTagPublisher
	database.Postgres.Where("publisher_id = ?", publisherID).Find(&publisherAdTags)

	publisherAdTagsShave := make(map[string]AdTagShave, len(publisherAdTags))
	for _, publisherTag := range publisherAdTags {
		publisherAdTagsShave[publisherTag.ID] = AdTagShave{
			Requests:    0,
			Impressions: 0,
		}
	}

	return publisherAdTagsShave
}

func calculateImpressionsShave(shave uint64, impressions int64, amount float64) (int64, float64) {
	if impressions > 1 {
		defaultAmount := (amount / float64(impressions)) * float64(1000)
		impressionsShaved := int64(float64(impressions) * (1.0 - float64(shave)/100.0))
		amountShaved := (float64(impressionsShaved) * defaultAmount) / float64(1000)
		return impressionsShaved, amountShaved
	} else {
		return impressions, amount
	}

}
func calculateRequestsShave(shave uint64, requests int64) int64 {
	if requests > 1 {
		return int64(float64(requests) * (1.0 - float64(shave)/100.0))
	} else {
		return requests
	}
}
