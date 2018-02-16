package admin

import (
	"bytes"
	"html/template"
	"net/http"

	"fmt"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	log "github.com/Sirupsen/logrus"
	"github.com/roistat/go-clickhouse"
)

func RtbStatisticsMainHandler(w http.ResponseWriter, r *http.Request) {
	fm1 := template.FuncMap{"calculateFillRate": calculateFillRate}
	fm2 := template.FuncMap{"commaSeparator": commaSeparator}

	t, _ := template.New("main").Funcs(fm1).Funcs(fm2).ParseFiles(
		"control/templates/main.html",
		"control/templates/statistics/rtb.html",
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
			t.ExecuteTemplate(w, "main", GetRtbStatsByParams(r))
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

type clickHouseRtbResponse struct {
	PublisherLink      string
	PublisherLinkID    string
	PublisherID        int32
	AdvertiserID       int32
	Publisher          string
	Advertiser         string
	Date               string
	DateTime           string
	PublisherPrice     int32
	BidResponse        int32
	BidResponseTime    int64
	BidResponseTimeout int32
	BidResponseEmpty   int32
	BidWin             int32
	BidFloorPrice      int32
	BidPrice           int32
	SecondPrice        int32
	Requests           int32
	Init               int32
	InitError          int32
	Auction            int32
	Impressions        int64
	Amount             float64
	Profit             float64
	FillRate           float64
	OriginalAmount     float64
	VPAIDLoaded        int32
	VPAIDStart         int32
	VPAIDAdRequested   int32
	VPAIDAdLoad        int32
	VPAIDAdError       int32
	VPAIDLoadPlayer    int32
	VPAIDPlayerLoaded  int32
	VPAIDBidReceived   int32
}

type responseStatisticsRtb struct {
	Advertisers        []models.Advertiser
	Publishers         []models.Publisher
	Stats              []clickHouseRtbResponse
	FieldsToShow       fieldsToShow
	AvailableTimezones []timezone
	StartDate          string
	EndDate            string
	PublisherID        int
	PublisherLinkID    string
	AdvertiserID       int

	//TotalStats         totalStats
	SelectedTimezone string
	OrderBy          OrderBy
}

func getRtbStatisticsByDates(startDate, endDate, selectedTimezone string) []clickHouseRtbResponse {
	queryString := fmt.Sprintf(`
		SELECT
			dictGetString('publisher_links', 'name', tuple(targeting_id)) as publisher_link,
			publisher_price,
			bid_requests,
			bid_response,
			bid_response_timeout,
			bid_response_empty,
			bid_win,
			date_with_timezone,
			init_count,
			init_error_count,
			auction_count,
			vpaid_loaded_count,
			vpaid_start_count,
			vpaid_ad_requested_count,
			vpaid_ad_load_count,
			vpaid_ad_error_count,
			//vpaid_load_player_count,
			//vpaid_player_loaded_count,
			//vpaid_bid_received_count,
			impressions,
			amount,
			origin_amount,
			profit
		FROM (
			SELECT
				targeting_id,
				publisher_price,
				sum(requests) as bid_requests,
				sum(bid_response) as bid_response,
				sum(bid_response_timeout) as bid_response_timeout,
				sum(bid_response_empty) as bid_response_empty,
				sum(bid_win) as bid_win,
				toDate(date_time, '%s') as date_with_timezone
			FROM statistics.rtb_bid_requests_merged
			WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s')
			GROUP BY targeting_id, publisher_price, date_with_timezone
		) ANY FULL OUTER JOIN (
			SELECT
				targeting_id,
				init_count,
				init_error_count,
				auction_count,
				vpaid_loaded_count,
				vpaid_start_count,
				vpaid_ad_requested_count,
				vpaid_ad_load_count,
				vpaid_ad_error_count,
				//vpaid_load_player_count,
				//vpaid_player_loaded_count,
				//vpaid_bid_received_count,
				publisher_price,
				impressions,
				amount,
				origin_amount,
				profit,
				date_with_timezone
			FROM (
				SELECT
					targeting_id,
						sumIf(events_count, event_name = 'init') as init_count,
						sumIf(events_count, event_name = 'init_error') as init_error_count,
						sumIf(events_count, event_name = 'auction') as auction_count,
						sumIf(events_count, event_name = 'vpaid_loaded') as vpaid_loaded_count,
						sumIf(events_count, event_name = 'vpaid_start') as vpaid_start_count,
						sumIf(events_count, event_name = 'vpaid_ad_requested') as vpaid_ad_requested_count,
						sumIf(events_count, event_name = 'vpaid_ad_load') as vpaid_ad_load_count,
						sumIf(events_count, event_name = 'vpaid_ad_error') as vpaid_ad_error_count,
						//sumIf(events_count, event_name = 'vpaid_load_player') as vpaid_load_player_count,
						//sumIf(events_count, event_name = 'vpaid_player_loaded') as vpaid_player_loaded_count,
						//sumIf(events_count, event_name = 'vpaid_bid_received') as vpaid_bid_received_count,
						publisher_price,
						toDate(date_time, '%s') as date_with_timezone
				FROM statistics.rtb_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s')
				GROUP BY targeting_id, publisher_price, date_with_timezone
			) ANY FULL OUTER JOIN (
				SELECT
					targeting_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit,
					toDate(date_time, '%s') as date_with_timezone
				FROM statistics.statistics_events_merged
				WHERE date_with_timezone >= toDate('%s') AND date_with_timezone <= toDate('%s')  AND event_name='impression' AND request_type = 'rtb'
				GROUP BY targeting_id, date_with_timezone
			) USING (targeting_id, date_with_timezone)
		) USING (targeting_id, publisher_price, date_with_timezone)
	`,
		selectedTimezone, startDate, endDate,
		selectedTimezone, startDate, endDate,
		selectedTimezone, startDate, endDate)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item clickHouseRtbResponse
	var stats []clickHouseRtbResponse

	for iter.Scan(
		&item.PublisherLink, &item.PublisherPrice, &item.Requests, &item.BidResponse,
		&item.BidResponseTimeout, &item.BidResponseEmpty, &item.BidWin, &item.Date, &item.Init, &item.InitError,
		&item.Auction, &item.VPAIDLoaded, &item.VPAIDStart, &item.VPAIDAdRequested, &item.VPAIDAdLoad, &item.VPAIDAdError,
		//&item.VPAIDLoadPlayer, &item.VPAIDPlayerLoaded, &item.VPAIDBidReceived,
		&item.Impressions, &item.Amount, &item.OriginalAmount, &item.Profit,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return stats
}

func GetRtbStatsByParams(r *http.Request) responseStatisticsRtb {
	var stats []clickHouseRtbResponse

	//total := totalStats{}
	visibleFields := fieldsToShow{}
	orderBy := OrderBy{}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	//advertiserID, _ := strconv.Atoi(r.URL.Query().Get("advertiser"))
	//publisherID, _ := strconv.Atoi(r.URL.Query().Get("publisher"))
	//adTagID, _ := strconv.Atoi(r.URL.Query().Get("ad_tag"))
	//adTagPublisherID := r.URL.Query().Get("ad_tag_publisher")
	selectedTimezone := r.URL.Query().Get("timezone")
	//groupBy := r.URL.Query().Get("group_by")
	orderByField := r.URL.Query().Get("order_by")
	orderByOrder := r.URL.Query().Get("order_by_order")
	pubLinkID := r.URL.Query().Get("pub_link")
	//domainsExport := r.URL.Query().Get("domains_export") == "true"
	//csvExport := r.URL.Query().Get("csv_export") == "true"
	//
	//splitBy := r.URL.Query().Get("split_by")
	//splitByValue, ok := splitByWhiteList[splitBy]

	//if ok == false {
	//	splitByValue = ""
	//}

	if pubLinkID == "" {
		pubLinkID = "0"
	}

	//if adTagID != 0 {
	//	adTagPublishers = createAdTagPublishersList(uint64(adTagID))
	//}

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

	//if domainsExport {
	//	stats = getStatisticsByAdTagAndPublisherAndDomain()
	//	visibleFields.Domain = true
	//	visibleFields.AdTag = true
	//	visibleFields.Publisher = true
	//} else if publisherID != 0 && adTagID != 0 && splitByValue != "" {
	//	stats = getStatisticsByAdTagAndPublisherSplited(startDate, endDate, adTagID, publisherID, selectedTimezone, splitByValue, orderBy, csvExport)
	//
	//	switch splitByValue {
	//	case "geo_country":
	//		visibleFields.GeoCountry = true
	//	case "domain":
	//		visibleFields.Domain = true
	//	case "device_type":
	//		visibleFields.DeviceType = true
	//	case "app_name":
	//		visibleFields.AppName = true
	//	case "bundle_id":
	//		visibleFields.BundleID = true
	//	}
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.AdTag = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = false
	//} else if advertiserID != 0 && adTagID != 0 && splitByValue != "" {
	//	stats = getStatisticsByAdTagSplited(startDate, endDate, adTagID, selectedTimezone, splitByValue, orderBy, csvExport)
	//
	//	switch splitByValue {
	//	case "geo_country":
	//		visibleFields.GeoCountry = true
	//	case "domain":
	//		visibleFields.Domain = true
	//	case "device_type":
	//		visibleFields.DeviceType = true
	//	case "app_name":
	//		visibleFields.AppName = true
	//	case "bundle_id":
	//		visibleFields.BundleID = true
	//	}
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.AdTag = true
	//	visibleFields.Date = false
	//} else if advertiserID != 0 && adTagID != 0 {
	//	stats = getStatisticsByAdTag(startDate, endDate, adTagID, selectedTimezone, orderBy)
	//	visibleFields.OrderBy = true
	//	visibleFields.AdTag = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = true
	//} else if adTagID != 0 && publisherID != 0 {
	//	stats = getStatisticsByAdTagAndPublisher(startDate, endDate, adTagID, publisherID, selectedTimezone)
	//	visibleFields.AdTag = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = true
	//	visibleFields.RequestType = true
	//} else if advertiserID != 0 && splitByValue != "" {
	//	stats = getStatisticsByAdvertiserSplited(startDate, endDate, advertiserID, selectedTimezone, splitByValue, orderBy, csvExport)
	//
	//	switch splitByValue {
	//	case "geo_country":
	//		visibleFields.GeoCountry = true
	//	case "domain":
	//		visibleFields.Domain = true
	//	case "device_type":
	//		visibleFields.DeviceType = true
	//	case "app_name":
	//		visibleFields.AppName = true
	//	case "bundle_id":
	//		visibleFields.BundleID = true
	//	}
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.Advertiser = true
	//	visibleFields.Date = false
	//} else if advertiserID != 0 && publisherID != 0 {
	//	stats = getStatisticsByAdvertiserAndPublisher(startDate, endDate, advertiserID, publisherID, selectedTimezone)
	//	visibleFields.Advertiser = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = true
	//} else if advertiserID != 0 {
	//	stats = getStatisticsByAdvertiser(startDate, endDate, advertiserID, selectedTimezone, orderBy)
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.Advertiser = true
	//	visibleFields.AdTag = true
	//	visibleFields.Date = true
	//} else if pubLinkID != "0" && splitByValue != "" {
	//	stats = getStatisticsByPublisherLinkSplited(startDate, endDate, pubLinkID, selectedTimezone, splitByValue, orderBy, csvExport)
	//
	//	switch splitByValue {
	//	case "geo_country":
	//		visibleFields.GeoCountry = true
	//	case "domain":
	//		visibleFields.Domain = true
	//	case "device_type":
	//		visibleFields.DeviceType = true
	//	case "app_name":
	//		visibleFields.AppName = true
	//	case "bundle_id":
	//		visibleFields.BundleID = true
	//	}
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.PublisherLink = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = true
	//	visibleFields.RequestType = true
	//} else if publisherID != 0 && splitByValue != "" {
	//	stats = getStatisticsByPublisherSplited(startDate, endDate, publisherID, selectedTimezone, splitByValue, orderBy, csvExport)
	//
	//	switch splitByValue {
	//	case "geo_country":
	//		visibleFields.GeoCountry = true
	//	case "domain":
	//		visibleFields.Domain = true
	//	case "device_type":
	//		visibleFields.DeviceType = true
	//	case "app_name":
	//		visibleFields.AppName = true
	//	case "bundle_id":
	//		visibleFields.BundleID = true
	//	}
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = false
	//} else if publisherID != 0 && pubLinkID == "0" {
	//	stats = getStatisticsByPublisher(startDate, endDate, publisherID, selectedTimezone, groupBy, orderBy)
	//
	//	groupByValue := groupByWhiteList[groupBy]
	//
	//	if groupByValue == "targeting_id" {
	//		visibleFields.PublisherLink = true
	//	} else {
	//		visibleFields.AdTag = true
	//	}
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.Publisher = true
	//	visibleFields.RequestType = true
	//	visibleFields.Date = true
	//} else if pubLinkID != "0" {
	//	stats = getStatisticsByPublisherLink(startDate, endDate, pubLinkID, selectedTimezone, orderBy)
	//	visibleFields.OrderBy = true
	//	visibleFields.PublisherLink = true
	//	visibleFields.AdTag = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = true
	//	visibleFields.RequestType = true
	//} else if adTagID != 0 && splitByValue != "" {
	//	stats = getStatisticsByAdTagSplited(startDate, endDate, adTagID, selectedTimezone, splitByValue, orderBy, csvExport)
	//
	//	switch splitByValue {
	//	case "geo_country":
	//		visibleFields.GeoCountry = true
	//	case "domain":
	//		visibleFields.Domain = true
	//	case "device_type":
	//		visibleFields.DeviceType = true
	//	case "app_name":
	//		visibleFields.AppName = true
	//	case "bundle_id":
	//		visibleFields.BundleID = true
	//	}
	//
	//	visibleFields.OrderBy = true
	//	visibleFields.AdTag = true
	//	visibleFields.RequestType = false
	//	visibleFields.Date = false
	//} else if adTagID != 0 {
	//	stats = getStatisticsByAdTag(startDate, endDate, adTagID, selectedTimezone, orderBy)
	//	visibleFields.OrderBy = true
	//	visibleFields.AdTag = true
	//	visibleFields.Publisher = true
	//	visibleFields.Date = true
	//} else if adTagPublisherID != "" {
	//	stats = getStatisticsByAdTagPublisher(startDate, endDate, adTagPublisherID, selectedTimezone)
	//	visibleFields.AdTagPublisher = true
	//	visibleFields.Date = true
	//	visibleFields.RequestType = true
	//} else {

	stats = getRtbStatisticsByDates(startDate, endDate, selectedTimezone)

	//visibleFields.Advertiser = true
	visibleFields.PublisherLink = true

	visibleFields.Date = true
	//}

	visibleFields.OrderBy = true
	//}

	//for _, item := range stats {
	//	total.Requests += item.Requests
	//	total.Impressions += item.Impressions
	//	total.Amount += item.Amount
	//	total.Profit += item.Profit
	//	total.OriginalAmount += item.OriginalAmount
	//}

	var advertisersList []models.Advertiser
	database.Postgres.Where("is_dsp = ?", "true").Find(&advertisersList)

	return responseStatisticsRtb{
		Advertisers:        advertisersList,
		Stats:              stats,
		FieldsToShow:       visibleFields,
		AvailableTimezones: AvailableTimezones,
		StartDate:          startDate,
		EndDate:            endDate,
		//PublisherID:        publisherID,
		//AdvertiserID:       advertiserID,
		PublisherLinkID: pubLinkID,
		//AdTagID:            adTagID,
		//TotalStats:         total,
		//AdTagPublishers:    adTagPublishers,
		SelectedTimezone: selectedTimezone,
		OrderBy:          orderBy,
	}
}
