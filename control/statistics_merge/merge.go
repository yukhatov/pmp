package statisticsMerge

import (
	"net/http"

	"time"

	"fmt"

	"strings"

	"bitbucket.org/tapgerine/pmp/control/database"
	log "github.com/Sirupsen/logrus"
	"github.com/roistat/go-clickhouse"
)

const (
	oneSecond      = 1
	tenMinutes     = 10 * 60
	fiveMinutes    = 5 * 60
	dateTimeFormat = "2006-01-02 15:04:05"
	dateFormat     = "2006-01-02"
)

func MergeClickHouseRequestStatistics(w http.ResponseWriter, r *http.Request) {
	lastMergeDateQuery := "SELECT max(date_time) FROM statistics.statistics_merged"

	query := clickhouse.NewQuery(lastMergeDateQuery)
	iter := query.Iter(database.ClickHouse)
	var lastMergeDate string
	iter.Scan(&lastMergeDate)

	if iter.Error() != nil {
		log.Warn(iter.Error())
		return
	}
	lastMergeDateParsed, _ := time.Parse(dateTimeFormat, lastMergeDate)
	currentTime := time.Now().UTC().Unix()

	if lastMergeDateParsed.Unix() == 1498227000 {
		lastMergeDateParsed = time.Unix(lastMergeDateParsed.Unix()-1, 0).UTC()
	}

	var isMergeNeeded bool
	var timesMerged int

	for {
		isMergeNeeded = (currentTime - tenMinutes) > lastMergeDateParsed.Unix()

		if !isMergeNeeded {
			break
		}
		mergeFrom := lastMergeDateParsed.Unix() + oneSecond
		mergeTo := lastMergeDateParsed.Unix() + fiveMinutes

		mergeFromDateTime := time.Unix(mergeFrom, 0).UTC()
		mergeToDateTime := time.Unix(mergeTo, 0).UTC()

		mergeQuery := fmt.Sprintf(`
			INSERT INTO statistics.statistics_merged (ad_tag_publisher_id, ad_tag_id, publisher_id, advertiser_id, requests, date, date_time, geo_country, device_type, request_type, targeting_id, domain, app_name, bundle_id)
			SELECT ad_tag_publisher_id, ad_tag_id, publisher_id, advertiser_id, toUInt32(count()), toDate('%s'), toDateTime('%s'), geo_country, device_type, request_type, targeting_id, domain, app_name, bundle_id
			FROM statistics.daily_statistics
			WHERE date = toDate('%s') AND date_time >= toDateTime('%s') AND date_time <= toDateTime('%s')
			GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, advertiser_id, geo_country, device_type, request_type, targeting_id, domain, app_name, bundle_id;`,
			mergeToDateTime.Format(dateFormat), mergeToDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateFormat),
			mergeFromDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateTimeFormat),
		)

		runningQueries := getRunningQueries()
		for _, query := range runningQueries {
			if strings.Contains(query, mergeToDateTime.Format(dateTimeFormat)) {
				// It means that previous query is still running, aborting
				log.Info(fmt.Sprintf("Previous query for %s is still running", mergeToDateTime.Format(dateTimeFormat)))
				return
			}
		}

		query = clickhouse.NewQuery(mergeQuery)
		err := query.Exec(database.ClickHouse)
		if err != nil {
			log.Warn(err)
			break
		}

		lastMergeDateParsed = mergeToDateTime
		timesMerged++
	}
	//log.Info(fmt.Sprintf("Merge succeded. Times merged: %d. Last merge time: %s", timesMerged, lastMergeDateParsed))
}

func MergeClickHouseEventsStatistics(w http.ResponseWriter, r *http.Request) {
	lastMergeDateQuery := "SELECT max(date_time) FROM statistics.statistics_events_merged"

	query := clickhouse.NewQuery(lastMergeDateQuery)
	iter := query.Iter(database.ClickHouse)
	var lastMergeDate string
	iter.Scan(&lastMergeDate)

	if iter.Error() != nil {
		log.Warn(iter.Error())
		return
	}

	if lastMergeDate == "" {
		lastMergeDate = "2017-07-04 13:04:59"
	}

	lastMergeDateParsed, _ := time.Parse(dateTimeFormat, lastMergeDate)
	currentTime := time.Now().UTC().Unix()

	var isMergeNeeded bool
	var timesMerged int

	for {
		isMergeNeeded = (currentTime - tenMinutes) > lastMergeDateParsed.Unix()

		if !isMergeNeeded {
			break
		}
		mergeFrom := lastMergeDateParsed.Unix() + oneSecond
		mergeTo := lastMergeDateParsed.Unix() + fiveMinutes

		mergeFromDateTime := time.Unix(mergeFrom, 0).UTC()
		mergeToDateTime := time.Unix(mergeTo, 0).UTC()

		mergeQuery := fmt.Sprintf(`
			INSERT INTO statistics.statistics_events_merged (ad_tag_publisher_id, ad_tag_id, publisher_id, advertiser_id, date, date_time, event_name, events_count, amount, origin_amount, request_type, geo_country, device_type, targeting_id, domain, app_name, bundle_id)
			SELECT ad_tag_publisher_id, ad_tag_id, publisher_id, advertiser_id, toDate('%s'), toDateTime('%s'), event_name, toUInt32(count()), toUInt32(sum(amount)), toUInt32(sum(origin_amount)), request_type, geo_country, device_type, targeting_id, domain, app_name, bundle_id
			FROM statistics.daily_statistics_events
			WHERE date = toDate('%s') AND date_time >= toDateTime('%s') AND date_time <= toDateTime('%s')
			GROUP BY ad_tag_publisher_id, ad_tag_id, publisher_id, advertiser_id, event_name, request_type, geo_country, device_type, targeting_id, domain, app_name, bundle_id;`,
			mergeToDateTime.Format(dateFormat), mergeToDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateFormat),
			mergeFromDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateTimeFormat),
		)

		query = clickhouse.NewQuery(mergeQuery)
		err := query.Exec(database.ClickHouse)
		if err != nil {
			log.Warn(err)
			break
		}

		lastMergeDateParsed = mergeToDateTime
		timesMerged++
	}
	//log.Info(fmt.Sprintf("Merge events succeded. Times merged: %d. Last merge time: %s", timesMerged, lastMergeDateParsed))
}

func MergeClickHouseRtbEventsStatistics(w http.ResponseWriter, r *http.Request) {
	lastMergeDateQuery := "SELECT max(date_time) FROM statistics.rtb_events_merged"

	query := clickhouse.NewQuery(lastMergeDateQuery)
	iter := query.Iter(database.ClickHouse)
	var lastMergeDate string
	iter.Scan(&lastMergeDate)

	if iter.Error() != nil {
		log.Warn(iter.Error())
		return
	}
	lastMergeDateParsed, _ := time.Parse(dateTimeFormat, lastMergeDate)
	currentTime := time.Now().UTC().Unix()

	if lastMergeDateParsed.Unix() < 0 {
		firstRecordDateQuery := "SELECT date FROM statistics.daily_rtb_events ORDER BY date ASC LIMIT 1"
		query = clickhouse.NewQuery(firstRecordDateQuery)
		iter = query.Iter(database.ClickHouse)
		var firstRecordDate string
		iter.Scan(&firstRecordDate)

		if iter.Error() != nil {
			log.Warn(iter.Error())
			return
		}

		lastMergeDateParsed, _ = time.Parse(dateFormat, firstRecordDate)
	}

	var isMergeNeeded bool
	var timesMerged int

	for {
		isMergeNeeded = (currentTime - tenMinutes) > lastMergeDateParsed.Unix()

		if !isMergeNeeded {
			break
		}
		mergeFrom := lastMergeDateParsed.Unix() + oneSecond
		mergeTo := lastMergeDateParsed.Unix() + fiveMinutes

		mergeFromDateTime := time.Unix(mergeFrom, 0).UTC()
		mergeToDateTime := time.Unix(mergeTo, 0).UTC()

		mergeQuery := fmt.Sprintf(`
			INSERT INTO statistics.rtb_events_merged (publisher_id, targeting_id, event_name, events_count, date, date_time, publisher_price, geo_country, device_type, domain, app_name, bundle_id)
			SELECT publisher_id, targeting_id, event_name, toUInt64(count()), toDate('%s'), toDateTime('%s'), publisher_price, geo_country, device_type, domain, app_name, bundle_id
			FROM statistics.daily_rtb_events
			WHERE date = toDate('%s') AND date_time >= toDateTime('%s') AND date_time <= toDateTime('%s')
			GROUP BY publisher_id, targeting_id, event_name, publisher_price, geo_country, device_type, domain, app_name, bundle_id;`,
			mergeToDateTime.Format(dateFormat), mergeToDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateFormat),
			mergeFromDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateTimeFormat),
		)

		runningQueries := getRunningQueries()
		for _, query := range runningQueries {
			if strings.Contains(query, mergeToDateTime.Format(dateTimeFormat)) {
				// It means that previous query is still running, aborting
				log.Info(fmt.Sprintf("Previous query for %s is still running", mergeToDateTime.Format(dateTimeFormat)))
				return
			}
		}

		query = clickhouse.NewQuery(mergeQuery)
		err := query.Exec(database.ClickHouse)
		if err != nil {
			log.Warn(err)
			break
		}

		lastMergeDateParsed = mergeToDateTime
		timesMerged++
	}
	//log.Info(fmt.Sprintf("Merge succeded. Times merged: %d. Last merge time: %s", timesMerged, lastMergeDateParsed))
}

func MergeClickHouseRtbRequestsStatistics(w http.ResponseWriter, r *http.Request) {
	currentTime := time.Now().UTC().Unix()

	lastMergeDateQuery := "SELECT max(date_time) FROM statistics.rtb_bid_requests_merged"

	query := clickhouse.NewQuery(lastMergeDateQuery)
	iter := query.Iter(database.ClickHouse)
	var lastMergeDate string
	iter.Scan(&lastMergeDate)

	if iter.Error() != nil {
		log.Warn(iter.Error())
		return
	}
	lastMergeDateParsed, _ := time.Parse(dateTimeFormat, lastMergeDate)

	if lastMergeDateParsed.Unix() < 0 {
		firstRecordDateQuery := "SELECT date FROM statistics.daily_rtb_bid_requests ORDER BY date ASC LIMIT 1"
		query = clickhouse.NewQuery(firstRecordDateQuery)
		iter = query.Iter(database.ClickHouse)
		var firstRecordDate string
		iter.Scan(&firstRecordDate)

		if iter.Error() != nil {
			log.Warn(iter.Error())
			return
		}

		lastMergeDateParsed, _ = time.Parse(dateFormat, firstRecordDate)
	}

	var isMergeNeeded bool
	var timesMerged int

	for {
		isMergeNeeded = (currentTime - tenMinutes) > lastMergeDateParsed.Unix()

		if !isMergeNeeded {
			break
		}
		mergeFrom := lastMergeDateParsed.Unix() + oneSecond
		mergeTo := lastMergeDateParsed.Unix() + fiveMinutes

		mergeFromDateTime := time.Unix(mergeFrom, 0).UTC()
		mergeToDateTime := time.Unix(mergeTo, 0).UTC()

		mergeQuery := fmt.Sprintf(`
			INSERT INTO statistics.rtb_bid_requests_merged (
				publisher_id, targeting_id, requests, date, date_time, publisher_price, advertiser_id, bid_response,
				bid_response_time_quantiles, bid_response_timeout, bid_response_empty, bid_win,
				bid_floor_price_quantiles, bid_price_quantiles, second_price_quantiles,
				geo_country, device_type, domain, app_name, bundle_id
			)
			SELECT
				publisher_id, targeting_id, toUInt64(count()) as requests, toDate('%s'), toDateTime('%s'), publisher_price,
				advertiser_id, toUInt32(sum(bid_response)),
				quantiles(0.25, 0.5, 0.75)(bid_response_time) as bid_response_time_quantiles,
				toUInt32(sum(bid_response_timeout)), toUInt32(sum(bid_response_empty)), toUInt32(sum(bid_win)),
				quantiles(0.25, 0.5, 0.75)(bid_floor_price) as bid_floor_price_quantiles,
				quantiles(0.25, 0.5, 0.75)(bid_price) as bid_price_quantiles,
				quantiles(0.25, 0.5, 0.75)(second_price) as second_price_quantiles,
				geo_country, device_type, domain, app_name, bundle_id
			FROM statistics.daily_rtb_bid_requests
			WHERE date = toDate('%s') AND date_time >= toDateTime('%s') AND date_time <= toDateTime('%s')
			GROUP BY publisher_id, targeting_id, advertiser_id, publisher_price, geo_country, device_type, domain, app_name, bundle_id;`,
			mergeToDateTime.Format(dateFormat), mergeToDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateFormat),
			mergeFromDateTime.Format(dateTimeFormat), mergeToDateTime.Format(dateTimeFormat),
		)

		runningQueries := getRunningQueries()
		for _, query := range runningQueries {
			if strings.Contains(query, mergeToDateTime.Format(dateTimeFormat)) {
				// It means that previous query is still running, aborting
				log.Info(fmt.Sprintf("Previous query for %s is still running", mergeToDateTime.Format(dateTimeFormat)))
				return
			}
		}

		query = clickhouse.NewQuery(mergeQuery)
		err := query.Exec(database.ClickHouse)
		if err != nil {
			log.Warn(err)
			break
		}

		lastMergeDateParsed = mergeToDateTime
		timesMerged++
	}
	//log.Info(fmt.Sprintf("Merge succeded. Times merged: %d. Last merge time: %s", timesMerged, lastMergeDateParsed))
}

func getRunningQueries() []string {
	checkQuery := "SELECT query FROM system.processes;"
	query := clickhouse.NewQuery(checkQuery)
	iter := query.Iter(database.ClickHouse)

	var item string
	runningQueries := []string{}
	for iter.Scan(
		&item,
	) {
		if item != "SELECT query FROM system.processes" {
			runningQueries = append(runningQueries, item)
		}
	}
	return runningQueries
}
