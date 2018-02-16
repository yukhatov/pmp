package admin

import (
	"html/template"
	"net/http"

	"fmt"

	"strconv"

	"bytes"
	"encoding/csv"

	"math"
	"strings"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	log "github.com/Sirupsen/logrus"
	"github.com/roistat/go-clickhouse"
)

type fieldsToShow struct {
	Advertiser     bool
	AdTag          bool
	Publisher      bool
	AdTagPublisher bool
	PublisherLink  bool
	RequestType    bool
	GeoCountry     bool
	DeviceType     bool
	Domain         bool
	Date           bool
	OrderBy        bool
	AppName        bool
	BundleID       bool
}

type totalStats struct {
	Requests       int64
	Impressions    int64
	Amount         float64
	OriginalAmount float64
	Profit         float64
}

type timezone struct {
	UserValue   string
	SystemValue string
}

var AvailableTimezones = []timezone{
	{UserValue: "UTC", SystemValue: "UTC"},
	{UserValue: "America/New_York", SystemValue: "America/New_York"},
	{UserValue: "Europe/Kiev", SystemValue: "Europe/Kiev"},
}

var groupByWhiteList = map[string]string{
	"ad_tag":                       "ad_tag_id",
	"advertiser":                   "advertiser_id",
	"publisher":                    "publisher_id",
	"ad_tag_publisher":             "ad_tag_publisher_id",
	"publisher_links":              "targeting_id",
	"publisher_links_with_ad_tags": "targeting_id, ad_tag_id",
}

var splitByWhiteList = map[string]string{
	"geo":       "geo_country",
	"device":    "device_type",
	"domain":    "domain",
	"app_name":  "app_name",
	"bundle_id": "bundle_id",
}

type responseStatistics struct {
	Advertisers        []models.Advertiser
	Publishers         []models.Publisher
	AdTags             []models.AdTag
	Stats              []clickHouseRequestsResponse
	FieldsToShow       fieldsToShow
	AvailableTimezones []timezone
	StartDate          string
	EndDate            string
	PublisherID        int
	PublisherLinkID    string
	AdvertiserID       int
	AdTagID            int
	AdTagPublisherID   string
	AdTagPublishers    []publishersListForAdTag
	TotalStats         totalStats
	SelectedTimezone   string
	OrderBy            OrderBy
}

type clickHouseRequestsResponse struct {
	AdTag            string
	PublisherLink    string
	PublisherLinkID  string
	PublisherID      int32
	AdvertiserID     int32
	AdTagID          int32
	AdTagPublisherID string
	AdTagPublisher   string
	Publisher        string
	Advertiser       string
	Requests         int64
	Date             string
	DateTime         string
	Impressions      int64
	Amount           float64
	Profit           float64
	FillRate         float64
	OriginalAmount   float64
	RequestType      string
	GeoCountry       string
	DeviceType       string
	Domain           string
	AppName          string
	BundleID         string
}

type OrderBy struct {
	field string
	order string
}

func calculateFillRate(impressions, requests int64) string {
	return fmt.Sprintf("%.4f", float64(impressions)/float64(requests)*100.0)
}

func commaSeparator(number int64) string {
	sign := ""

	// minin64 can't be negated to a usable value, so it has to be special cased.
	if number == math.MinInt64 {
		return "-9,223,372,036,854,775,808"
	}

	if number < 0 {
		sign = "-"
		number = 0 - number
	}

	parts := []string{"", "", "", "", "", "", ""}
	j := len(parts) - 1

	for number > 999 {
		parts[j] = strconv.FormatInt(number%1000, 10)
		switch len(parts[j]) {
		case 2:
			parts[j] = "0" + parts[j]
		case 1:
			parts[j] = "00" + parts[j]
		}
		number = number / 1000
		j--
	}
	parts[j] = strconv.Itoa(int(number))

	return sign + strings.Join(parts[j:], " ")
}

func getStatisticsByDates(startDate, endDate, selectedTimezone, groupBy string, orderBy OrderBy) []clickHouseRequestsResponse {
	groupByValue, ok := groupByWhiteList[groupBy]
	getStringParameter := "toUInt64(%s)"

	if ok == false {
		groupByValue = "ad_tag_id"
		groupBy = "ad_tag"
	}

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	var name = &item.AdTag

	if groupByValue == "advertiser_id" {
		name = &item.Advertiser
	} else if groupByValue == "publisher_id" {
		name = &item.Publisher
	} else if groupByValue == "ad_tag_publisher_id" {
		name = &item.AdTagPublisher
		getStringParameter = "tuple(%s)"
	} else if groupByValue == "targeting_id" {
		name = &item.PublisherLink
		getStringParameter = "tuple(%s)"
	}

	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('%s', 'name', `+getStringParameter+`), concat(' - ',toString(%s))) as name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			%s
		FROM (
			SELECT
				%s,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_publisher_id != 'GhQbRtYa'
			GROUP BY %s, date_with_timezone
		)
		ANY FULL OUTER JOIN (
			SELECT
				%s,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				event_requests
			FROM (
				SELECT
					%s,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_publisher_id != 'GhQbRtYa' AND event_name='request'
				GROUP BY %s, date_with_timezone
			)
			ANY FULL OUTER JOIN (
				SELECT
					%s,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_publisher_id != 'GhQbRtYa' AND event_name='impression'
				GROUP BY %s, date_with_timezone
			) USING (%s, date_with_timezone)
		) USING (%s, date_with_timezone)
		`,
		groupBy, groupByValue, groupByValue, groupByValue, groupByValue, selectedTimezone, startDate, endDate, groupByValue,
		groupByValue, groupByValue, selectedTimezone, startDate, endDate, groupByValue, groupByValue, selectedTimezone, startDate, endDate, groupByValue, groupByValue, groupByValue)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY date_with_timezone", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var id string

	for iter.Scan(
		name, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount, &item.OriginalAmount,
		&item.Profit, &item.Date, &id,
	) {
		if groupByValue == "advertiser_id" {
			intId, _ := strconv.Atoi(id)
			item.AdvertiserID = int32(intId)
		} else if groupByValue == "publisher_id" {
			intId, _ := strconv.Atoi(id)
			item.PublisherID = int32(intId)
		} else if groupByValue == "ad_tag_publisher_id" {
			item.AdTagPublisherID = id
		} else if groupByValue == "targeting_id" {
			item.PublisherLinkID = id
		} else {
			intId, _ := strconv.Atoi(id)
			item.AdTagID = int32(intId)
		}

		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return stats
}

func getStatisticsByDatesWithGroupByLinkAndAdTag(startDate, endDate, selectedTimezone string, orderBy OrderBy) []clickHouseRequestsResponse {
	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse

	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('publisher_links', 'name', tuple(targeting_id)), concat(' - ',toString(targeting_id))) as publisher_link,
			dictGetString('ad_tag', 'name', dictGetUInt64('ad_tag_publisher', 'ad_tag_id', tuple(ad_tag_publisher_id))) as ad_tag_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit
		FROM (
			SELECT
				ad_tag_publisher_id,
				ad_tag_id,
				targeting_id,
				sum(requests) as requests
			FROM statistics.statistics_merged
			WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND request_type = 'targeting'
			GROUP BY ad_tag_publisher_id, ad_tag_id, targeting_id
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_publisher_id,
				ad_tag_id,
				targeting_id,
				impressions,
				amount,
				origin_amount,
				profit,
				event_requests
			FROM (
				SELECT
					ad_tag_publisher_id,
					ad_tag_id,
					targeting_id,
					sum(events_count) as event_requests
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND event_name='request' AND request_type = 'targeting'
				GROUP BY ad_tag_publisher_id, ad_tag_id, targeting_id
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_publisher_id,
					ad_tag_id,
					targeting_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND event_name='impression' AND request_type = 'targeting'
				GROUP BY ad_tag_publisher_id, ad_tag_id, targeting_id
			) USING (ad_tag_publisher_id, ad_tag_id, targeting_id)
		) USING (ad_tag_publisher_id, ad_tag_id, targeting_id)
		WHERE publisher_link != ''
		`,
		selectedTimezone, startDate, selectedTimezone, endDate,
		selectedTimezone, startDate, selectedTimezone, endDate,
		selectedTimezone, startDate, selectedTimezone, endDate)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY publisher_link, ad_tag_name", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	for iter.Scan(
		&item.PublisherLink, &item.AdTag, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount, &item.OriginalAmount,
		&item.Profit,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return stats
}

func getStatisticsByAdvertiser(startDate, endDate string, advertiserID int, selectedTimezone string, orderBy OrderBy) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('advertiser', 'name', toUInt64(advertiser_id)), concat(' - ',toString(advertiser_id))) as advertiser_name,
			concat(dictGetString('ad_tag', 'name', toUInt64(ad_tag_id)), concat(' - ',toString(ad_tag_id))) as ad_tag_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			advertiser_id,
			ad_tag_id
		FROM (
			SELECT
				advertiser_id,
				ad_tag_id,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND advertiser_id = toUInt32(%d)
			GROUP BY advertiser_id, date_with_timezone, ad_tag_id
		)
		ANY FULL OUTER JOIN (
			SELECT
				advertiser_id,
				ad_tag_id,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				event_requests
			FROM (
				SELECT
					advertiser_id,
					ad_tag_id,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND advertiser_id = toUInt32(%d) AND event_name='request'
				GROUP BY advertiser_id, date_with_timezone, ad_tag_id
			)
			ANY FULL OUTER JOIN (
				SELECT
					advertiser_id,
					ad_tag_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND advertiser_id = toUInt32(%d) AND event_name='impression'
				GROUP BY advertiser_id, date_with_timezone, ad_tag_id
			) USING (advertiser_id, ad_tag_id, date_with_timezone)
		) USING (advertiser_id, ad_tag_id, date_with_timezone)
		`,
		selectedTimezone, startDate, endDate, advertiserID,
		selectedTimezone, startDate, endDate, advertiserID,
		selectedTimezone, startDate, endDate, advertiserID,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY date_with_timezone", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.Advertiser, &item.AdTag, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount, &item.OriginalAmount,
		&item.Profit, &item.Date, &item.AdvertiserID, &item.AdTagID,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdvertiserSplited(startDate, endDate string, advertiserID int, selectedTimezone string, splitBy string, orderBy OrderBy, csvExport bool) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('advertiser', 'name', toUInt64(advertiser_id)), concat(' - ',toString(advertiser_id))) as advertiser_name,
			requests,
			impressions,
			(impressions / requests) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			%s,
			advertiser_id
		FROM (
			SELECT
				advertiser_id,
				sum(requests) as requests,
				%s
			FROM statistics.statistics_merged
			WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND advertiser_id = toUInt32(%d)
			GROUP BY advertiser_id, %s
		)
		ANY FULL OUTER JOIN (
			SELECT
				advertiser_id,
				impressions,
				amount,
				origin_amount,
				profit,
				%s
			FROM (
				SELECT
					advertiser_id,
					sum(events_count) as event_requests,
					%s
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND advertiser_id = toUInt32(%d) AND event_name='request'
				GROUP BY advertiser_id, %s
			)
			ANY FULL OUTER JOIN (
				SELECT
					advertiser_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					%s
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND advertiser_id = toUInt32(%d) AND event_name='impression'
				GROUP BY advertiser_id, %s
			) USING (advertiser_id, %s)
		) USING (advertiser_id, %s)
		`,
		splitBy, splitBy, selectedTimezone, startDate, selectedTimezone, endDate, advertiserID, splitBy, splitBy, splitBy,
		selectedTimezone, startDate, selectedTimezone, endDate, advertiserID, splitBy,
		splitBy, selectedTimezone, startDate, selectedTimezone, endDate, advertiserID,
		splitBy, splitBy, splitBy,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY requests desc", queryString)
	}

	if !csvExport {
		queryString = fmt.Sprintf("%s LIMIT 1000", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	var splitedBy string

	for iter.Scan(
		&item.Advertiser, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount, &item.OriginalAmount,
		&item.Profit, &splitedBy, &item.AdvertiserID,
	) {
		switch splitBy {
		case "geo_country":
			item.GeoCountry = splitedBy
		case "domain":
			item.Domain = splitedBy
		case "device_type":
			item.DeviceType = splitedBy
		case "app_name":
			item.AppName = splitedBy
		case "bundle_id":
			item.BundleID = splitedBy
		}

		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByPublisher(startDate, endDate string, publisherID int, selectedTimezone string, groupBy string, orderBy OrderBy) []clickHouseRequestsResponse {
	groupByValue, ok := groupByWhiteList[groupBy]
	getStringParameter := "toUInt64(%s)"

	if ok == false {
		groupByValue = "ad_tag_id"
		groupBy = "ad_tag"
	}

	var item clickHouseRequestsResponse
	var name = &item.AdTag

	if groupByValue == "targeting_id" {
		name = &item.PublisherLink
		getStringParameter = "tuple(%s)"
	}

	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
			dictGetString('%s', 'name', `+getStringParameter+`) as name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			publisher_id,
			%s,
			request_type
		FROM (
			SELECT
				publisher_id,
				%s,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone,
				request_type
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND publisher_id = toUInt32(%d)
			GROUP BY publisher_id, date_with_timezone, request_type, %s
		)
		ANY FULL OUTER JOIN (
			SELECT
				publisher_id,
				%s,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				request_type,
				event_requests
			FROM (
				SELECT
					publisher_id,
					%s,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone,
					request_type
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND publisher_id = toUInt32(%d) AND event_name='request'
				GROUP BY publisher_id, date_with_timezone, request_type, %s
			) ANY FULL OUTER JOIN (
				SELECT
					publisher_id,
					%s,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone,
					request_type
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND publisher_id = toUInt32(%d) AND event_name='impression'
				GROUP BY publisher_id, date_with_timezone, request_type, %s
			) USING (publisher_id, %s, date_with_timezone, request_type)
		) USING (publisher_id, %s, date_with_timezone, request_type)
		`,
		groupBy, groupByValue, groupByValue, groupByValue, selectedTimezone, startDate, endDate, publisherID, groupByValue, groupByValue, groupByValue,
		selectedTimezone, startDate, endDate, publisherID, groupByValue, groupByValue,
		selectedTimezone, startDate, endDate, publisherID, groupByValue, groupByValue, groupByValue,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY date_with_timezone", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)
	var id string

	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.Publisher, name, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount, &item.OriginalAmount,
		&item.Profit, &item.Date, &item.PublisherID, &id, &item.RequestType,
	) {
		if groupByValue == "targeting_id" {
			item.PublisherLinkID = id
		} else {
			intId, _ := strconv.Atoi(id)
			item.AdTagID = int32(intId)
		}

		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByPublisherSplited(startDate, endDate string, publisherID int, selectedTimezone string, splitBy string, orderBy OrderBy, csvExport bool) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			%s,
			publisher_id
		FROM (
			SELECT
			  publisher_id,
			  sum(requests) as requests,
			  %s
			FROM statistics.statistics_merged
			WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND publisher_id = toUInt32(%d)
			GROUP BY publisher_id, %s
		)
		ANY LEFT JOIN (
			SELECT
				publisher_id,
				impressions,
				amount,
				origin_amount,
				profit,
				event_requests,
				%s
			FROM (
				SELECT
					publisher_id,
					sum(events_count) as event_requests,
					%s
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND publisher_id = toUInt32(%d) AND event_name='request'
				GROUP BY publisher_id, %s
			)
			ANY FULL OUTER JOIN (
				SELECT
					publisher_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					%s
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND publisher_id = toUInt32(%d) AND event_name='impression'
				GROUP BY publisher_id, %s
			) USING (publisher_id, %s)
		) USING (publisher_id, %s)
		`,
		splitBy, splitBy, selectedTimezone, startDate, selectedTimezone, endDate, publisherID, splitBy, splitBy, splitBy,
		selectedTimezone, startDate, selectedTimezone, endDate, publisherID, splitBy, splitBy,
		selectedTimezone, startDate, selectedTimezone, endDate, publisherID, splitBy, splitBy, splitBy,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY requests desc", queryString)
	}

	if !csvExport {
		queryString = fmt.Sprintf("%s LIMIT 1000", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	var splitedBy string

	for iter.Scan(
		&item.Publisher, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount, &item.OriginalAmount,
		&item.Profit, &splitedBy, &item.PublisherID,
	) {
		switch splitBy {
		case "geo_country":
			item.GeoCountry = splitedBy
		case "domain":
			item.Domain = splitedBy
		case "device_type":
			item.DeviceType = splitedBy
		case "app_name":
			item.AppName = splitedBy
		case "bundle_id":
			item.BundleID = splitedBy
		}

		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdvertiserAndPublisher(startDate, endDate string, advertiserID, publisherID int, selectedTimezone string) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('advertiser', 'name', toUInt64(advertiser_id)), concat(' - ',toString(advertiser_id))) as advertiser_name,
			concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			advertiser_id,
			publisher_id
		FROM (
			SELECT
				advertiser_id,
				publisher_id,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND publisher_id = toUInt32(%d) AND advertiser_id = toUInt32(%d)
			GROUP BY advertiser_id, publisher_id, date_with_timezone
		)
		ANY FULL OUTER JOIN (
			SELECT
				advertiser_id,
				publisher_id,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				event_requests
			FROM (
				SELECT
					advertiser_id,
					publisher_id,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND publisher_id = toUInt32(%d) AND advertiser_id = toUInt32(%d) AND event_name='request'
				GROUP BY advertiser_id, publisher_id, date_with_timezone
			)
			ANY FULL OUTER JOIN (
				SELECT
					advertiser_id,
					publisher_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND publisher_id = toUInt32(%d) AND advertiser_id = toUInt32(%d) AND event_name='impression'
				GROUP BY advertiser_id, publisher_id, date_with_timezone
			) USING (advertiser_id, publisher_id, date_with_timezone)
		) USING (advertiser_id, publisher_id, date_with_timezone)
		ORDER BY date_with_timezone`,
		selectedTimezone, startDate, endDate, publisherID, advertiserID,
		selectedTimezone, startDate, endDate, publisherID, advertiserID,
		selectedTimezone, startDate, endDate, publisherID, advertiserID,
	)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.Advertiser, &item.Publisher, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.Date, &item.AdvertiserID, &item.PublisherID,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdTagAndPublisherSplited(startDate, endDate string, adTagID int, adTagPublisherID int, selectedTimezone string, splitBy string, orderBy OrderBy, csvExport bool) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			dictGetString('ad_tag', 'name', dictGetUInt64('ad_tag_publisher', 'ad_tag_id', tuple(ad_tag_publisher_id))) as ad_tag_name,
			concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			publisher_id,
			ad_tag_id,
			%s
		FROM (
			SELECT
				ad_tag_publisher_id,
				publisher_id,
				ad_tag_id,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone,
				%s
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND publisher_id = toUInt32(%d)
			GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, date_with_timezone, %s
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_publisher_id,
				publisher_id,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				event_requests,
				%s
			FROM (
				SELECT
					ad_tag_publisher_id,
					publisher_id,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone,
					%s
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND publisher_id = toUInt32(%d) AND event_name='request'
				GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, date_with_timezone, %s
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_publisher_id,
					publisher_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone,
					%s
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND publisher_id = toUInt32(%d) AND event_name='impression'
				GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, date_with_timezone, %s
			) USING (ad_tag_publisher_id, publisher_id, date_with_timezone, %s)
		) USING (ad_tag_publisher_id, publisher_id, date_with_timezone, %s)
		`,
		splitBy, selectedTimezone, splitBy, startDate, endDate, adTagID, adTagPublisherID, splitBy, splitBy,
		selectedTimezone, splitBy, startDate, endDate, adTagID, adTagPublisherID, splitBy,
		selectedTimezone, splitBy,
		startDate, endDate, adTagID, adTagPublisherID, splitBy, splitBy, splitBy,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY date_with_timezone", queryString)
	}

	if !csvExport {
		queryString = fmt.Sprintf("%s LIMIT 1000", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	var splitedBy string

	for iter.Scan(
		&item.AdTag, &item.Publisher, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.Date, &item.PublisherID, &item.AdTagID, &splitedBy,
	) {
		switch splitBy {
		case "geo_country":
			item.GeoCountry = splitedBy
		case "domain":
			item.Domain = splitedBy
		case "device_type":
			item.DeviceType = splitedBy
		case "app_name":
			item.AppName = splitedBy
		case "bundle_id":
			item.BundleID = splitedBy
		}

		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdTagPublisher(startDate, endDate string, adTagPublisherID string, selectedTimezone string) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			dictGetString('ad_tag_publisher', 'name', tuple(ad_tag_publisher_id)) as name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			ad_tag_publisher_id
		FROM (
			SELECT
				ad_tag_publisher_id,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_publisher_id = '%s'
			GROUP BY ad_tag_publisher_id, date_with_timezone
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_publisher_id,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				event_requests
			FROM (
				SELECT
					ad_tag_publisher_id,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_publisher_id = '%s' AND event_name='request'
				GROUP BY ad_tag_publisher_id, date_with_timezone
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_publisher_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_publisher_id = '%s' AND event_name='impression'
				GROUP BY ad_tag_publisher_id, date_with_timezone
			) USING (ad_tag_publisher_id, date_with_timezone)
		) USING (ad_tag_publisher_id, date_with_timezone)
		ORDER BY date_with_timezone`,
		selectedTimezone, startDate, endDate, adTagPublisherID,
		selectedTimezone, startDate, endDate, adTagPublisherID,
		selectedTimezone, startDate, endDate, adTagPublisherID,
	)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.AdTagPublisher, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.Date, &item.AdTagPublisherID,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByPublisherLinkSplited(startDate, endDate string, pubLinkID string, selectedTimezone string, splitBy string, orderBy OrderBy, csvExport bool) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
			SELECT
				concat(dictGetString('publisher_links', 'name', tuple(targeting_id)), concat(' - ',toString(targeting_id))) as name,
				concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
				requests + event_requests,
				impressions,
				(impressions / (requests + event_requests)) * 100 as fill_rate,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				targeting_id,
				publisher_id,
				request_type,
				%s
			FROM (
				SELECT
					publisher_id,
					targeting_id,
					sum(requests) as requests,
					toDate(date_time, '%s') as date_with_timezone,
					request_type,
					%s
				FROM statistics.statistics_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND targeting_id = '%s'
				GROUP BY targeting_id, publisher_id, date_with_timezone, request_type, %s
			)
			ANY FULL OUTER JOIN (
				SELECT
					publisher_id,
					impressions,
					amount,
					origin_amount,
					profit,
					date_with_timezone,
					request_type,
					event_requests,
					%s
				FROM (
					SELECT
						publisher_id,
						sum(events_count) as event_requests,
						toDate(date_time, '%s') as date_with_timezone,
						request_type,
						%s
					FROM statistics.statistics_events_merged
					WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND targeting_id = '%s' AND event_name='request'
					GROUP BY targeting_id, publisher_id, date_with_timezone, request_type, %s
				)
				ANY FULL OUTER JOIN (
					SELECT
						publisher_id,
						sum(events_count) as impressions,
						sum(amount) / 1000000 as amount,
						sum(origin_amount) / 1000000 as origin_amount,
						(origin_amount - amount) as profit,
						toDate(date_time, '%s') as date_with_timezone,
						request_type,
						%s
					FROM statistics.statistics_events_merged
					WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND targeting_id = '%s' AND event_name='impression'
					GROUP BY targeting_id, publisher_id, date_with_timezone, request_type, %s
				) USING (publisher_id, date_with_timezone, request_type, %s)
			) USING (publisher_id, date_with_timezone, request_type, %s)
			`,
		splitBy, selectedTimezone, splitBy, startDate, endDate, pubLinkID, splitBy,
		splitBy, selectedTimezone, splitBy, startDate, endDate, pubLinkID, splitBy,
		selectedTimezone, splitBy, startDate, endDate, pubLinkID, splitBy, splitBy, splitBy,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY date_with_timezone", queryString)
	}

	if !csvExport {
		queryString = fmt.Sprintf("%s LIMIT 1000", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)
	var splitedBy string

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.PublisherLink, &item.Publisher, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.Date, &item.PublisherLinkID, &item.PublisherID, &item.RequestType, &splitedBy,
	) {
		switch splitBy {
		case "geo_country":
			item.GeoCountry = splitedBy
		case "domain":
			item.Domain = splitedBy
		case "device_type":
			item.DeviceType = splitedBy
		case "app_name":
			item.AppName = splitedBy
		case "bundle_id":
			item.BundleID = splitedBy
		}

		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByPublisherLink(startDate, endDate string, pubLinkID string, selectedTimezone string, orderBy OrderBy) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
			SELECT
				concat(dictGetString('publisher_links', 'name', tuple(targeting_id)), concat(' - ',toString(targeting_id))) as name,
				concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
				concat(dictGetString('ad_tag', 'name', toUInt64(ad_tag_id)), concat(' - ',toString(ad_tag_id))) as ad_tag_name,
				requests + event_requests,
				impressions,
				(impressions / (requests + event_requests)) * 100 as fill_rate,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				targeting_id,
				publisher_id,
				ad_tag_id,
				request_type
			FROM (
				SELECT
					publisher_id,
					targeting_id,
					ad_tag_id,
					sum(requests) as requests,
					toDate(date_time, '%s') as date_with_timezone,
					request_type
				FROM statistics.statistics_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND targeting_id = '%s'
				GROUP BY ad_tag_id, targeting_id, publisher_id, date_with_timezone, request_type
			)
			ANY FULL OUTER JOIN (
				SELECT
					publisher_id,
					ad_tag_id,
					impressions,
					amount,
					origin_amount,
					profit,
					date_with_timezone,
					request_type,
					event_requests
				FROM (
					SELECT
						publisher_id,
						ad_tag_id,
						sum(events_count) as event_requests,
						toDate(date_time, '%s') as date_with_timezone,
						request_type
					FROM statistics.statistics_events_merged
					WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND targeting_id = '%s' AND event_name='request'
					GROUP BY ad_tag_id, targeting_id, publisher_id, date_with_timezone, request_type
				)
				ANY FULL OUTER JOIN (
					SELECT
						publisher_id,
						ad_tag_id,
						sum(events_count) as impressions,
						sum(amount) / 1000000 as amount,
						sum(origin_amount) / 1000000 as origin_amount,
						(origin_amount - amount) as profit,
						toDate(date_time, '%s') as date_with_timezone,
						request_type
					FROM statistics.statistics_events_merged
					WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND targeting_id = '%s' AND event_name='impression'
					GROUP BY ad_tag_id, targeting_id, publisher_id, date_with_timezone, request_type
				) USING (ad_tag_id, publisher_id, date_with_timezone, request_type)
			) USING (ad_tag_id, publisher_id, date_with_timezone, request_type)
			`,
		selectedTimezone, startDate, endDate, pubLinkID,
		selectedTimezone, startDate, endDate, pubLinkID,
		selectedTimezone, startDate, endDate, pubLinkID,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY date_with_timezone", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.PublisherLink, &item.Publisher, &item.AdTag, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.Date, &item.PublisherLinkID, &item.PublisherID, &item.AdTagID, &item.RequestType,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdTagAndPublisherAndDomain() []clickHouseRequestsResponse {
	queryString := `
		SELECT
			dictGetString('ad_tag', 'name', dictGetUInt64('ad_tag_publisher', 'ad_tag_id', tuple(ad_tag_publisher_id))) as ad_tag_name,
			concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
			requests + event_requests,
			impressions,
			publisher_id,
			ad_tag_id,
			domain
		FROM (
			SELECT
				ad_tag_publisher_id,
				publisher_id,
				ad_tag_id,
				sum(requests) as requests,
				domain
			FROM statistics.statistics_merged
			WHERE date >= today() - 30
			GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, domain
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_publisher_id,
				publisher_id,
				impressions,
				domain,
				event_requests
			FROM (
				SELECT
					ad_tag_publisher_id,
					publisher_id,
					sum(events_count) as event_requests,
					domain
				FROM statistics.statistics_events_merged
				WHERE date >= today() - 30 AND event_name='request'
				GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, domain
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_publisher_id,
					publisher_id,
					sum(events_count) as impressions,
					domain
				FROM statistics.statistics_events_merged
				WHERE date >= today() - 30 AND event_name='impression'
				GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, domain
			) USING (ad_tag_publisher_id, publisher_id, domain)
		) USING (ad_tag_publisher_id, publisher_id, domain)
		ORDER BY publisher_name, ad_tag_name`

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.AdTag, &item.Publisher, &item.Requests, &item.Impressions, &item.PublisherID, &item.AdTagID, &item.Domain,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdTagAndPublisher(startDate, endDate string, adTagID int, adTagPublisherID int, selectedTimezone string) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			dictGetString('ad_tag', 'name', dictGetUInt64('ad_tag_publisher', 'ad_tag_id', tuple(ad_tag_publisher_id))) as ad_tag_name,
			concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			publisher_id,
			ad_tag_id,
			request_type
		FROM (
			SELECT
				ad_tag_publisher_id,
				publisher_id,
				ad_tag_id,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone,
				request_type
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND publisher_id = toUInt32(%d)
			GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, date_with_timezone, request_type
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_publisher_id,
				publisher_id,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				request_type,
				event_requests
			FROM (
				SELECT
					ad_tag_publisher_id,
					publisher_id,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone,
					request_type
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND publisher_id = toUInt32(%d) AND event_name='request'
				GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, date_with_timezone, request_type
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_publisher_id,
					publisher_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone,
					request_type
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND publisher_id = toUInt32(%d) AND event_name='impression'
				GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, date_with_timezone, request_type
			) USING (ad_tag_publisher_id, publisher_id, date_with_timezone, request_type)
		) USING (ad_tag_publisher_id, publisher_id, date_with_timezone, request_type)
		ORDER BY date_with_timezone`,
		selectedTimezone, startDate, endDate, adTagID, adTagPublisherID,
		selectedTimezone, startDate, endDate, adTagID, adTagPublisherID,
		selectedTimezone, startDate, endDate, adTagID, adTagPublisherID,
	)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.AdTag, &item.Publisher, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.Date, &item.PublisherID, &item.AdTagID, &item.RequestType,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdTag(startDate, endDate string, adTagID int, selectedTimezone string, orderBy OrderBy) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('ad_tag', 'name', toUInt64(ad_tag_id)), concat(' - ',toString(ad_tag_id))) as ad_tag_name,
			concat(dictGetString('publisher', 'name', toUInt64(publisher_id)), concat(' - ',toString(publisher_id))) as publisher_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			date_with_timezone,
			ad_tag_id,
			publisher_id
		FROM (
			SELECT
				ad_tag_id,
				publisher_id,
				sum(requests) as requests,
				toDate(date_time, '%s') as date_with_timezone
			FROM statistics.statistics_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d)
			GROUP BY ad_tag_id, date_with_timezone, publisher_id
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_id,
				publisher_id,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone,
				event_requests
			FROM (
				SELECT
					ad_tag_id,
					publisher_id,
					sum(events_count) as event_requests,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND event_name='request'
				GROUP BY ad_tag_id, date_with_timezone, publisher_id
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_id,
					publisher_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND event_name='impression'
				GROUP BY ad_tag_id, date_with_timezone, publisher_id
			) USING (ad_tag_id, date_with_timezone, publisher_id)
		) USING (ad_tag_id, date_with_timezone, publisher_id)
		`,
		selectedTimezone, startDate, endDate, adTagID,
		selectedTimezone, startDate, endDate, adTagID,
		selectedTimezone, startDate, endDate, adTagID,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY date_with_timezone", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	for iter.Scan(
		&item.AdTag, &item.Publisher, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.Date, &item.AdTagID, &item.PublisherID,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func getStatisticsByAdTagSplited(startDate, endDate string, adTagID int, selectedTimezone string, splitBy string, orderBy OrderBy, csvExport bool) []clickHouseRequestsResponse {
	queryString := fmt.Sprintf(`
		SELECT
			concat(dictGetString('ad_tag', 'name', toUInt64(ad_tag_id)), concat(' - ',toString(ad_tag_id))) as ad_tag_name,
			requests + event_requests,
			impressions,
			(impressions / (requests + event_requests)) * 100 as fill_rate,
			amount,
			origin_amount,
			profit,
			ad_tag_id,
			%s
		FROM (
			SELECT
				ad_tag_id,
				sum(requests) as requests,
				%s
			FROM statistics.statistics_merged
			WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND ad_tag_id = toUInt32(%d)
			GROUP BY ad_tag_id, %s
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_id,
				impressions,
				amount,
				origin_amount,
				profit,
				event_requests,
				%s
			FROM (
				SELECT
					ad_tag_id,
					sum(events_count) as event_requests,
					%s
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND event_name='request'
				GROUP BY ad_tag_id, %s
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					%s
				FROM statistics.statistics_events_merged
				WHERE toDate(date_time, '%s') >= toDate('%s') AND toDate(date_time, '%s') <= toDate('%s') AND ad_tag_id = toUInt32(%d) AND event_name='impression'
				GROUP BY ad_tag_id, %s
			) USING (ad_tag_id, %s)
		) USING (ad_tag_id, %s)
		`,
		splitBy, splitBy, selectedTimezone, startDate, selectedTimezone, endDate, adTagID, splitBy,
		splitBy, splitBy, selectedTimezone, startDate, selectedTimezone, endDate, adTagID,
		splitBy, splitBy, selectedTimezone, startDate, selectedTimezone, endDate, adTagID,
		splitBy, splitBy, splitBy,
	)

	if orderBy.field != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s %s", queryString, orderBy.field, orderBy.order)
	} else {
		queryString = fmt.Sprintf("%s ORDER BY requests desc", queryString)
	}

	if !csvExport {
		queryString = fmt.Sprintf("%s LIMIT 1000", queryString)
	}

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRequestsResponse
	var stats []clickHouseRequestsResponse
	var splitedBy string

	for iter.Scan(
		&item.AdTag, &item.Requests, &item.Impressions, &item.FillRate, &item.Amount,
		&item.OriginalAmount, &item.Profit, &item.AdTagID, &splitedBy,
	) {

		switch splitBy {
		case "geo_country":
			item.GeoCountry = splitedBy
		case "domain":
			item.Domain = splitedBy
		case "device_type":
			item.DeviceType = splitedBy
		case "app_name":
			item.AppName = splitedBy
		case "bundle_id":
			item.BundleID = splitedBy
		}

		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}
	return stats
}

func StatisticsMainHandler(w http.ResponseWriter, r *http.Request) {
	fm1 := template.FuncMap{"calculateFillRate": calculateFillRate}
	fm2 := template.FuncMap{"commaSeparator": commaSeparator}

	t, _ := template.New("main").Funcs(fm1).Funcs(fm2).ParseFiles(
		"control/templates/main.html",
		"control/templates/statistics/main.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)

	if len(r.URL.Query()) > 0 {
		/*
			Statistic with params
		*/
		csvExport := r.URL.Query().Get("csv_export") == "true"
		domainsExport := r.URL.Query().Get("domains_export") == "true"
		if csvExport || domainsExport {
			stats := GetStatsByParams(r)
			b := &bytes.Buffer{}
			createCSVDocument(b, stats.FieldsToShow, stats.Stats)

			w.Header().Set("Content-Type", "text/csv")
			if csvExport {
				w.Header().Set("Content-Disposition", "attachment;filename=Statistics.csv")
			} else if domainsExport {
				w.Header().Set("Content-Disposition", "attachment;filename=Domains.csv")
			}
			w.Write(b.Bytes())
		} else {
			t.ExecuteTemplate(w, "main", GetStatsByParams(r))
		}

	} else {
		/*
			Plain statistic
		*/
		t.ExecuteTemplate(w, "main", responseStatistics{
			AvailableTimezones: AvailableTimezones,
		})
	}
}

func GetStatsByParams(r *http.Request) responseStatistics {
	var stats []clickHouseRequestsResponse
	var adTagPublishers []publishersListForAdTag
	total := totalStats{}
	visibleFields := fieldsToShow{}
	orderBy := OrderBy{}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	advertiserID, _ := strconv.Atoi(r.URL.Query().Get("advertiser"))
	publisherID, _ := strconv.Atoi(r.URL.Query().Get("publisher"))
	adTagID, _ := strconv.Atoi(r.URL.Query().Get("ad_tag"))
	adTagPublisherID := r.URL.Query().Get("ad_tag_publisher")
	selectedTimezone := r.URL.Query().Get("timezone")
	groupBy := r.URL.Query().Get("group_by")
	orderByField := r.URL.Query().Get("order_by")
	orderByOrder := r.URL.Query().Get("order_by_order")
	pubLinkID := r.URL.Query().Get("pub_link")
	domainsExport := r.URL.Query().Get("domains_export") == "true"
	csvExport := r.URL.Query().Get("csv_export") == "true"

	splitBy := r.URL.Query().Get("split_by")
	splitByValue, ok := splitByWhiteList[splitBy]

	if ok == false {
		splitByValue = ""
	}

	if pubLinkID == "" {
		pubLinkID = "0"
	}

	if adTagID != 0 {
		adTagPublishers = createAdTagPublishersList(uint64(adTagID))
	}

	if selectedTimezone == "" {
		selectedTimezone = AvailableTimezones[0].SystemValue
	}

	if orderByField != "" {
		orderBy.field = orderByField
		if orderByOrder != "" {
			orderBy.order = orderByOrder
		} else {
			orderBy.order = "desc"
		}
	}

	if domainsExport {
		stats = getStatisticsByAdTagAndPublisherAndDomain()
		visibleFields.Domain = true
		visibleFields.AdTag = true
		visibleFields.Publisher = true
	} else if publisherID != 0 && adTagID != 0 && splitByValue != "" {
		stats = getStatisticsByAdTagAndPublisherSplited(startDate, endDate, adTagID, publisherID, selectedTimezone, splitByValue, orderBy, csvExport)

		switch splitByValue {
		case "geo_country":
			visibleFields.GeoCountry = true
		case "domain":
			visibleFields.Domain = true
		case "device_type":
			visibleFields.DeviceType = true
		case "app_name":
			visibleFields.AppName = true
		case "bundle_id":
			visibleFields.BundleID = true
		}

		visibleFields.OrderBy = true
		visibleFields.AdTag = true
		visibleFields.Publisher = true
		visibleFields.Date = false
	} else if advertiserID != 0 && adTagID != 0 && splitByValue != "" {
		stats = getStatisticsByAdTagSplited(startDate, endDate, adTagID, selectedTimezone, splitByValue, orderBy, csvExport)

		switch splitByValue {
		case "geo_country":
			visibleFields.GeoCountry = true
		case "domain":
			visibleFields.Domain = true
		case "device_type":
			visibleFields.DeviceType = true
		case "app_name":
			visibleFields.AppName = true
		case "bundle_id":
			visibleFields.BundleID = true
		}

		visibleFields.OrderBy = true
		visibleFields.AdTag = true
		visibleFields.Date = false
	} else if advertiserID != 0 && adTagID != 0 {
		stats = getStatisticsByAdTag(startDate, endDate, adTagID, selectedTimezone, orderBy)
		visibleFields.OrderBy = true
		visibleFields.AdTag = true
		visibleFields.Publisher = true
		visibleFields.Date = true
	} else if adTagID != 0 && publisherID != 0 {
		stats = getStatisticsByAdTagAndPublisher(startDate, endDate, adTagID, publisherID, selectedTimezone)
		visibleFields.AdTag = true
		visibleFields.Publisher = true
		visibleFields.Date = true
		visibleFields.RequestType = true
	} else if advertiserID != 0 && splitByValue != "" {
		stats = getStatisticsByAdvertiserSplited(startDate, endDate, advertiserID, selectedTimezone, splitByValue, orderBy, csvExport)

		switch splitByValue {
		case "geo_country":
			visibleFields.GeoCountry = true
		case "domain":
			visibleFields.Domain = true
		case "device_type":
			visibleFields.DeviceType = true
		case "app_name":
			visibleFields.AppName = true
		case "bundle_id":
			visibleFields.BundleID = true
		}

		visibleFields.OrderBy = true
		visibleFields.Advertiser = true
		visibleFields.Date = false
	} else if advertiserID != 0 && publisherID != 0 {
		stats = getStatisticsByAdvertiserAndPublisher(startDate, endDate, advertiserID, publisherID, selectedTimezone)
		visibleFields.Advertiser = true
		visibleFields.Publisher = true
		visibleFields.Date = true
	} else if advertiserID != 0 {
		stats = getStatisticsByAdvertiser(startDate, endDate, advertiserID, selectedTimezone, orderBy)

		visibleFields.OrderBy = true
		visibleFields.Advertiser = true
		visibleFields.AdTag = true
		visibleFields.Date = true
	} else if pubLinkID != "0" && splitByValue != "" {
		stats = getStatisticsByPublisherLinkSplited(startDate, endDate, pubLinkID, selectedTimezone, splitByValue, orderBy, csvExport)

		switch splitByValue {
		case "geo_country":
			visibleFields.GeoCountry = true
		case "domain":
			visibleFields.Domain = true
		case "device_type":
			visibleFields.DeviceType = true
		case "app_name":
			visibleFields.AppName = true
		case "bundle_id":
			visibleFields.BundleID = true
		}

		visibleFields.OrderBy = true
		visibleFields.PublisherLink = true
		visibleFields.Publisher = true
		visibleFields.Date = true
		visibleFields.RequestType = true
	} else if publisherID != 0 && splitByValue != "" {
		stats = getStatisticsByPublisherSplited(startDate, endDate, publisherID, selectedTimezone, splitByValue, orderBy, csvExport)

		switch splitByValue {
		case "geo_country":
			visibleFields.GeoCountry = true
		case "domain":
			visibleFields.Domain = true
		case "device_type":
			visibleFields.DeviceType = true
		case "app_name":
			visibleFields.AppName = true
		case "bundle_id":
			visibleFields.BundleID = true
		}

		visibleFields.OrderBy = true
		visibleFields.Publisher = true
		visibleFields.Date = false
	} else if publisherID != 0 && pubLinkID == "0" {
		stats = getStatisticsByPublisher(startDate, endDate, publisherID, selectedTimezone, groupBy, orderBy)

		groupByValue := groupByWhiteList[groupBy]

		if groupByValue == "targeting_id" {
			visibleFields.PublisherLink = true
		} else {
			visibleFields.AdTag = true
		}

		visibleFields.OrderBy = true
		visibleFields.Publisher = true
		visibleFields.RequestType = true
		visibleFields.Date = true
	} else if pubLinkID != "0" {
		stats = getStatisticsByPublisherLink(startDate, endDate, pubLinkID, selectedTimezone, orderBy)
		visibleFields.OrderBy = true
		visibleFields.PublisherLink = true
		visibleFields.AdTag = true
		visibleFields.Publisher = true
		visibleFields.Date = true
		visibleFields.RequestType = true
	} else if adTagID != 0 && splitByValue != "" {
		stats = getStatisticsByAdTagSplited(startDate, endDate, adTagID, selectedTimezone, splitByValue, orderBy, csvExport)

		switch splitByValue {
		case "geo_country":
			visibleFields.GeoCountry = true
		case "domain":
			visibleFields.Domain = true
		case "device_type":
			visibleFields.DeviceType = true
		case "app_name":
			visibleFields.AppName = true
		case "bundle_id":
			visibleFields.BundleID = true
		}

		visibleFields.OrderBy = true
		visibleFields.AdTag = true
		visibleFields.RequestType = false
		visibleFields.Date = false
	} else if adTagID != 0 {
		stats = getStatisticsByAdTag(startDate, endDate, adTagID, selectedTimezone, orderBy)
		visibleFields.OrderBy = true
		visibleFields.AdTag = true
		visibleFields.Publisher = true
		visibleFields.Date = true
	} else if adTagPublisherID != "" {
		stats = getStatisticsByAdTagPublisher(startDate, endDate, adTagPublisherID, selectedTimezone)
		visibleFields.AdTagPublisher = true
		visibleFields.Date = true
		visibleFields.RequestType = true
	} else {
		if groupBy == "publisher_links_with_ad_tags" {
			stats = getStatisticsByDatesWithGroupByLinkAndAdTag(startDate, endDate, selectedTimezone, orderBy)
			visibleFields.PublisherLink = true
			visibleFields.AdTag = true
		} else {
			stats = getStatisticsByDates(startDate, endDate, selectedTimezone, groupBy, orderBy)
			groupByValue := groupByWhiteList[groupBy]

			if groupByValue == "advertiser_id" {
				visibleFields.Advertiser = true
			} else if groupByValue == "publisher_id" {
				visibleFields.Publisher = true
			} else if groupByValue == "ad_tag_publisher_id" {
				visibleFields.AdTagPublisher = true
			} else if groupByValue == "targeting_id" {
				visibleFields.PublisherLink = true
			} else {
				visibleFields.AdTag = true
			}

			visibleFields.Date = true
		}

		visibleFields.OrderBy = true
	}

	for _, item := range stats {
		total.Requests += item.Requests
		total.Impressions += item.Impressions
		total.Amount += item.Amount
		total.Profit += item.Profit
		total.OriginalAmount += item.OriginalAmount
	}

	return responseStatistics{
		Stats:              stats,
		FieldsToShow:       visibleFields,
		AvailableTimezones: AvailableTimezones,
		StartDate:          startDate,
		EndDate:            endDate,
		PublisherID:        publisherID,
		AdvertiserID:       advertiserID,
		PublisherLinkID:    pubLinkID,
		AdTagID:            adTagID,
		TotalStats:         total,
		AdTagPublishers:    adTagPublishers,
		SelectedTimezone:   selectedTimezone,
		OrderBy:            orderBy,
	}
}

func createCSVDocument(b *bytes.Buffer, visibleFields fieldsToShow, stats []clickHouseRequestsResponse) {
	csvWriter := csv.NewWriter(b)

	header := []string{}
	if visibleFields.Advertiser {
		header = append(header, "Advertiser")
	}
	if visibleFields.AdTag {
		header = append(header, "Ad Tag")
	}
	if visibleFields.Publisher {
		header = append(header, "Publisher")
	}
	if visibleFields.Date {
		header = append(header, "Date")
	}
	if visibleFields.RequestType {
		header = append(header, "Request type")
	}
	if visibleFields.Domain {
		header = append(header, "Domain")
	}
	if visibleFields.PublisherLink {
		header = append(header, "Publisher's source")
	}

	if visibleFields.GeoCountry {
		header = append(header, "Country")
	}

	if visibleFields.AppName {
		header = append(header, "App Name")
	}
	if visibleFields.BundleID {
		header = append(header, "Bundle ID")
	}
	if visibleFields.DeviceType {
		header = append(header, "Device")
	}
	if visibleFields.AdTagPublisher {
		header = append(header, "Publisher's URL")
	}

	header = append(header, "Requests")
	header = append(header, "Impressions")
	header = append(header, "Fill rate")
	header = append(header, "PubRev")
	header = append(header, "AdvRev")
	header = append(header, "Profit")

	csvWriter.Write(header)

	for _, item := range stats {
		element := []string{}
		if visibleFields.Advertiser {
			element = append(element, item.Advertiser)
		}
		if visibleFields.AdTag {
			element = append(element, item.AdTag)
		}
		if visibleFields.Publisher {
			element = append(element, item.Publisher)
		}
		if visibleFields.Date {
			element = append(element, item.Date)
		}
		if visibleFields.RequestType {
			element = append(element, item.RequestType)
		}
		if visibleFields.GeoCountry {
			element = append(element, item.GeoCountry)
		}
		if visibleFields.Domain {
			element = append(element, item.Domain)
		}
		if visibleFields.PublisherLink {
			element = append(element, item.PublisherLink)
		}
		if visibleFields.AppName {
			element = append(element, item.AppName)
		}
		if visibleFields.BundleID {
			element = append(element, item.BundleID)
		}
		if visibleFields.DeviceType {
			element = append(element, item.DeviceType)
		}
		if visibleFields.AdTagPublisher {
			element = append(element, item.AdTagPublisher)
		}

		element = append(element, fmt.Sprintf("%d", item.Requests))
		element = append(element, fmt.Sprintf("%d", item.Impressions))
		element = append(element, calculateFillRate(item.Impressions, item.Requests))
		element = append(element, fmt.Sprintf("%.2f", item.Amount))
		element = append(element, fmt.Sprintf("%.2f", item.OriginalAmount))
		element = append(element, fmt.Sprintf("%.2f", item.Profit))
		csvWriter.Write(element)
	}

	csvWriter.Flush()
}
