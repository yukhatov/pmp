package dataSync

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"

	"encoding/json"

	"fmt"
	"strings"

	"math"

	log "github.com/Sirupsen/logrus"
	"github.com/roistat/go-clickhouse"
)

type SyncData struct {
	AdTags                       map[string]AdTagData                               `json:"ad_tags"`
	ParametersMapping            map[uint64]map[string]map[string]ParametersMapping `json:"parameters_mapping"`
	OurPlatformParametersMapping map[string]map[string]map[uint64]ParametersMapping `json:"our_platform_parameters_mapping"`
	PublisherTargetingIDMap      map[string]uint64                                  `json:"publisher_targeting_id_map"`
	TargetingLinkAdTagsIDs       map[string][]string                                `json:"targeting_link_ad_tags_i_ds"`
	PublisherLinks               map[string]PublisherLinkData                       `json:"publisher_links"`
	Advertisers                  map[uint64]AdvertiserData                          `json:"advertisers"`
}

type AdvertiserData struct {
	ID                uint64 `json:"id"`
	RTBIntegrationUrl string `json:"rtb_url"`
}

type AdTagData struct {
	AdTagID                  uint64                 `json:"id"`
	URL                      string                 `json:"url"`
	SupportsVast             bool                   `json:"supports_vast"`
	IsActive                 bool                   `json:"is_active"`
	IsAdTagPubActive         bool                   `json:"is_ad_tag_pub_active"`
	IsTest                   bool                   `json:"is_test"`
	AdvertiserPlatformTypeID uint64                 `json:"advertiser_platform_type_id"`
	Targeting                AdTagTargeting         `json:"targeting"`
	PublisherID              uint64                 `json:"publisher_id"`
	PublisherTargetingID     string                 `json:"publisher_targeting_id"`
	Price                    float64                `json:"price"`
	CouldBeUsedForTargeting  bool                   `json:"could_be_used_for_targeting"`
	ERPRByGeoForLastWeek     map[string]ERPRData    `json:"erpr_by_geo_for_last_week"`
	TotalStats               TotalStatsForLastMonth `json:"total_stats"`
	ERPRByTargetingID        map[string]ERPRData    `json:"erpr_by_targeting_id"`
	// TODO: use new compact format for FillRateByTargetingIDAndDomain (ERPRData)
	FillRateByTargetingIDAndDomain map[string]map[string]ERPRData `json:"fill_rate_by_domain"`
	DomainsListID                  uint64                         `json:"domains_list_id"`
	DomainsListType                string                         `json:"domains_list_type"`
}

type PublisherLinkData struct {
	ID              string
	DomainsListID   uint64
	DomainsListType string
	Platform        string
	Price           float64
	Optimization    string
	StudyRequests   int64
}

type AdTagTargeting struct {
	Geo        []string `json:"geo_targeting"`
	DeviceType string   `json:"device_type"`
}

type ParametersMapping struct {
	Shortcut         string `json:"shortcut"`
	Macros           string `json:"macros"`
	OriginalShortcut string `json:"original_shortcut"`
	OriginalMacros   string `json:"original_macros"`
	IsRequired       bool   `json:"is_required"`
	Platform         string `json:"platform"`
}

type ERPRDataByAdTagPubID struct {
	AdTagPublisherID string  `json:"-"`
	GeoCountry       string  `json:"geo_country"`
	Requests         int64   `json:"requests"`
	Impressions      int64   `json:"impressions"`
	FillRate         float64 `json:"fill_rate"`
	Margin           float64 `json:"margin"`
}

type ERPRData struct {
	Requests    int64   `json:"requests"`
	Impressions int64   `json:"impressions"`
	FillRate    float64 `json:"fill_rate"`
	Margin      float64 `json:"margin"`
	ERPR        float64 `json:"erpr"`
}

type ERPRDataByTargetingID struct {
	AdTagPublisherID string  `json:"-"`
	TargetingID      string  `json:"targeting_id"`
	Requests         int64   `json:"requests"`
	Impressions      int64   `json:"impressions"`
	FillRate         float64 `json:"fill_rate"`
	Margin           float64 `json:"margin"`
	Domain           string  `json:"domain"`
}

type TotalStatsForLastMonth struct {
	Impressions int64
	Requests    int64
}

func SyncDataIntoRedis(w http.ResponseWriter, r *http.Request) {
	var adTagPubList []*models.AdTagPublisher
	var advertiserList []*models.Advertiser

	database.Postgres.
		Preload("AdTag").
		Preload("Publisher").
		Preload("AdTag.Advertiser").
		Preload("AdTag.DomainsList").
		Where("is_active = ?", "true").
		Find(&adTagPubList)

	database.Postgres.
		Where("is_dsp = ?", "true").
		Find(&advertiserList)

	var syncData SyncData
	publisherUrlIDs := []string{}
	syncData.AdTags = make(map[string]AdTagData)
	syncData.Advertisers = make(map[uint64]AdvertiserData)
	syncData.ParametersMapping = make(map[uint64]map[string]map[string]ParametersMapping)

	for _, item := range advertiserList {
		syncData.Advertisers[item.ID] = AdvertiserData{
			ID:                item.ID,
			RTBIntegrationUrl: item.RTBIntegrationUrl,
		}
	}

	for _, item := range adTagPubList {
		if item.AdTag.IsArchived == true || item.AdTag.IsActive == false {
			continue
		}

		publisherUrlIDs = append(publisherUrlIDs, fmt.Sprintf("'%s'", item.ID))
		syncData.AdTags[item.ID] = AdTagData{
			AdTagID:          item.AdTagID,
			IsAdTagPubActive: item.IsActive,
			URL:              item.AdTag.URL,
			SupportsVast:     item.AdTag.IsVast,
			IsActive:         item.AdTag.IsActive,
			IsTest:           item.Publisher.Name == "Test",
			AdvertiserPlatformTypeID: item.AdTag.AdvertiserPlatformTypeID,
			Targeting: AdTagTargeting{
				Geo:        item.AdTag.GeoCountry,
				DeviceType: item.AdTag.DeviceType,
			},
			PublisherID:          item.PublisherID,
			PublisherTargetingID: item.Publisher.TargetingID,
			Price:                item.Price,
			CouldBeUsedForTargeting:        item.AdTag.IsTargeted,
			ERPRByGeoForLastWeek:           make(map[string]ERPRData),
			ERPRByTargetingID:              make(map[string]ERPRData),
			FillRateByTargetingIDAndDomain: make(map[string]map[string]ERPRData),
			DomainsListID:                  item.AdTag.DomainsListID,
			DomainsListType:                item.AdTag.DomainsList.Type,
		}
	}

	publisherUrlIDsJoined := strings.Join(publisherUrlIDs, ",")
	erprForLastWeek := getPublisherURLsERPRForLastWeek(publisherUrlIDsJoined)

	for _, erprItem := range erprForLastWeek {
		var margin float64
		if math.IsNaN(erprItem.Margin) {
			margin = 0.0
		} else {
			margin = erprItem.Margin
		}
		syncData.AdTags[erprItem.AdTagPublisherID].ERPRByGeoForLastWeek[erprItem.GeoCountry] = ERPRData{
			Requests:    erprItem.Requests,
			Impressions: erprItem.Impressions,
			FillRate:    erprItem.FillRate,
			Margin:      margin,
		}
	}

	totalStatsForLastMonth := getPublisherURLsTotalStatsForLastMonth(publisherUrlIDsJoined)
	for pubURLID, totalItem := range totalStatsForLastMonth {
		t := syncData.AdTags[pubURLID].TotalStats
		t.Requests = totalItem.Requests
		t.Impressions = totalItem.Impressions
	}

	var advertiserPlatforms []*models.AdvertiserPlatformTypes
	database.Postgres.Preload("ParametersMaps").Preload("ParametersMaps.OriginalParameter").Find(&advertiserPlatforms)

	for _, platform := range advertiserPlatforms {
		parameters := make(map[string]map[string]ParametersMapping)
		for _, parameter := range platform.ParametersMaps {
			if _, ok := parameters[parameter.Shortcut]; !ok {
				parameters[parameter.Shortcut] = make(map[string]ParametersMapping)
			}
			parameters[parameter.Shortcut][parameter.OriginalParameter.Platform] = ParametersMapping{
				Shortcut:         parameter.Shortcut,
				Macros:           parameter.Macros,
				OriginalShortcut: parameter.OriginalParameter.Shortcut,
				OriginalMacros:   parameter.OriginalParameter.Macros,
				IsRequired:       parameter.IsRequired,
				Platform:         parameter.OriginalParameter.Platform,
			}
		}
		syncData.ParametersMapping[platform.ID] = parameters
	}

	syncData.OurPlatformParametersMapping = make(map[string]map[string]map[uint64]ParametersMapping)
	for platformID, parameters := range syncData.ParametersMapping {
		for _, platform := range parameters {
			for _, parameter := range platform {
				if _, ok := syncData.OurPlatformParametersMapping[parameter.OriginalShortcut]; !ok {
					syncData.OurPlatformParametersMapping[parameter.OriginalShortcut] = make(map[string]map[uint64]ParametersMapping)
				}
				if _, ok := syncData.OurPlatformParametersMapping[parameter.OriginalShortcut][parameter.Platform]; !ok {
					syncData.OurPlatformParametersMapping[parameter.OriginalShortcut][parameter.Platform] = make(map[uint64]ParametersMapping)
				}
				syncData.OurPlatformParametersMapping[parameter.OriginalShortcut][parameter.Platform][platformID] = parameter
			}
		}
	}

	//////////////////////////////////////
	// Sync for old publisher dynamic links
	//////////////////////////////////////
	var publishers []*models.Publisher
	var publisherTargetingIDMap map[string]uint64

	database.Postgres.Find(&publishers)
	publisherTargetingIDMap = make(map[string]uint64, len(publishers))

	for _, pubItem := range publishers {
		publisherTargetingIDMap[pubItem.TargetingID] = pubItem.ID
	}

	//////////////////////////////////////
	// Sync for new publisher dynamic links
	//////////////////////////////////////
	var publisherLinks []models.PublisherLink
	database.Postgres.Preload("PublisherLinkAdTagPublisher").Preload("DomainsList").Find(&publisherLinks)

	var publisherLinksData map[string]PublisherLinkData
	publisherLinksData = make(map[string]PublisherLinkData, len(publisherLinks))
	for _, link := range publisherLinks {
		publisherTargetingIDMap[link.ID] = link.PublisherID
		publisherLinksData[link.ID] = PublisherLinkData{
			ID:              link.ID,
			DomainsListID:   link.DomainsListID,
			DomainsListType: link.DomainsList.Type,
			Platform:        link.Platform,
			Price:           link.Price,
			Optimization:    link.Optimization,
			StudyRequests:   int64(link.StudyRequests),
		}
	}
	syncData.PublisherTargetingIDMap = publisherTargetingIDMap
	syncData.PublisherLinks = publisherLinksData

	// Getting publisher links for statistics request
	var publisherLinksIDs []string
	publisherLinksIDs = make([]string, len(publisherLinks))
	for i, link := range publisherLinks {
		publisherLinksIDs[i] = link.ID
	}

	// Syncing ad tag ids list for every publisher link
	var publisherLinkAdTagMap map[string][]string
	publisherLinkAdTagMap = make(map[string][]string, len(publisherLinks))

	var adTagPublisherIDs []string
	for _, link := range publisherLinks {
		adTagPublisherIDs = make([]string, len(link.PublisherLinkAdTagPublisher))
		for i, item := range link.PublisherLinkAdTagPublisher {
			if item.IsActive {
				adTagPublisherIDs[i] = item.AdTagPublisherID
			}
		}
		publisherLinkAdTagMap[link.ID] = adTagPublisherIDs
	}
	syncData.TargetingLinkAdTagsIDs = publisherLinkAdTagMap

	for _, erprItem := range getPublisherLinksStatsByAdTag(publisherLinksIDs) {
		var margin float64
		if math.IsNaN(erprItem.Margin) {
			margin = 0.0
		} else {
			margin = erprItem.Margin
		}

		_, exists := syncData.AdTags[erprItem.AdTagPublisherID]
		if exists {
			syncData.AdTags[erprItem.AdTagPublisherID].ERPRByTargetingID[erprItem.TargetingID] = ERPRData{
				Requests:    erprItem.Requests,
				Impressions: erprItem.Impressions,
				FillRate:    erprItem.FillRate,
				Margin:      margin,
				ERPR:        erprItem.FillRate * margin,
			}
		}
	}

	// Collecting erpr and domains for ad tags in pub sources
	for _, erprItem := range getPublisherLinksStatsByAdTagAndDomain(publisherLinksIDs) {
		_, exists := syncData.AdTags[erprItem.AdTagPublisherID]
		if exists {

			_, exists = syncData.AdTags[erprItem.AdTagPublisherID].FillRateByTargetingIDAndDomain[erprItem.TargetingID]
			if !exists {
				syncData.AdTags[erprItem.AdTagPublisherID].FillRateByTargetingIDAndDomain[erprItem.TargetingID] = make(map[string]ERPRData)
			}
			if erprItem.Domain != "" && erprItem.FillRate > 0 {
				syncData.AdTags[erprItem.AdTagPublisherID].FillRateByTargetingIDAndDomain[erprItem.TargetingID][erprItem.Domain] = ERPRData{
					FillRate: erprItem.FillRate,
				}
			}
		}
	}

	jsonData, err := json.Marshal(syncData)
	if err != nil {
		log.Warn(err)
		return
	}

	database.Redis.Set("serving_data", jsonData, 0)
	database.Redis2.Set("serving_data", jsonData, 0)

	SyncDomainsLists()
}

func SyncDomainsLists() {
	var domainsLists []models.DomainsList
	database.Postgres.Find(&domainsLists)

	var domainsForRedis map[string]interface{}
	for _, list := range domainsLists {
		domainsForRedis = make(map[string]interface{}, len(list.Domains))
		for _, domain := range list.Domains {
			domainsForRedis[domain] = list.Type
		}
		database.Redis.Del(fmt.Sprintf("domains:%d", list.ID))
		database.Redis.HMSet(fmt.Sprintf("domains:%d", list.ID), domainsForRedis)
		database.Redis2.Del(fmt.Sprintf("domains:%d", list.ID))
		database.Redis2.HMSet(fmt.Sprintf("domains:%d", list.ID), domainsForRedis)
	}
}

func getPublisherURLsTotalStatsForLastMonth(publisherUrlIDs string) map[string]TotalStatsForLastMonth {
	queryString := fmt.Sprintf(`
		SELECT
			ad_tag_publisher_id,
			requests,
			impressions
		FROM (
			SELECT
				ad_tag_publisher_id,
				sum(requests) as requests
			FROM statistics.statistics_merged
			WHERE date >= today() - 30 AND ad_tag_publisher_id in (%s)
			GROUP BY ad_tag_publisher_id
		)
		ANY LEFT JOIN (
			SELECT
				ad_tag_publisher_id,
				sum(events_count) as impressions
			FROM statistics.statistics_events_merged
			WHERE date >= today() - 30 AND ad_tag_publisher_id in (%s)
			GROUP BY ad_tag_publisher_id
		) USING (ad_tag_publisher_id)`, publisherUrlIDs, publisherUrlIDs)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item TotalStatsForLastMonth
	var pubURLID string
	var stats map[string]TotalStatsForLastMonth
	stats = make(map[string]TotalStatsForLastMonth)

	for iter.Scan(
		&pubURLID, &item.Requests, &item.Impressions,
	) {
		stats[pubURLID] = item
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return stats
}

func getPublisherURLsERPRForLastWeek(publisherUrlIDs string) []ERPRDataByAdTagPubID {
	queryString := fmt.Sprintf(`
		SELECT
			ad_tag_publisher_id,
			geo_country,
			requests,
			impressions,
			if (toUInt8(requests), (impressions / requests) * 100, 0) as fill_rate,
			if (toUInt8(impressions), (origin_amount - amount) / impressions * 1000, 0) as margin
		FROM (
			SELECT
				ad_tag_publisher_id,
				geo_country,
				sum(requests) as requests
			FROM statistics.statistics_merged
			WHERE date >= today() - 7 AND ad_tag_publisher_id in (%s)
			GROUP BY ad_tag_publisher_id, geo_country
		)
		ANY LEFT JOIN (
			SELECT
				ad_tag_publisher_id,
				geo_country,
				sum(events_count) as impressions,
				sum(amount) / 1000000 as amount,
				sum(origin_amount) / 1000000 as origin_amount
			FROM statistics.statistics_events_merged
			WHERE date >= today() - 7 AND ad_tag_publisher_id in (%s)
			GROUP BY ad_tag_publisher_id, geo_country
		) USING (ad_tag_publisher_id, geo_country)`, publisherUrlIDs, publisherUrlIDs)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item ERPRDataByAdTagPubID
	var stats []ERPRDataByAdTagPubID
	for iter.Scan(
		&item.AdTagPublisherID, &item.GeoCountry, &item.Requests, &item.Impressions, &item.FillRate, &item.Margin,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return stats
}

func getPublisherLinksStatsByAdTagAndDomain(publisherLinksIDs []string) []ERPRDataByTargetingID {
	var linksIDsString []string
	linksIDsString = make([]string, len(publisherLinksIDs))

	for i, id := range publisherLinksIDs {
		linksIDsString[i] = fmt.Sprintf("'%s'", id)
	}

	linksIDsJoined := strings.Join(linksIDsString, ",")

	queryString := fmt.Sprintf(`
		SELECT
			ad_tag_publisher_id,
			targeting_id,
			if (toUInt8(requests), (impressions / requests) * 100, 0) as fill_rate,
			domain
		FROM (
			SELECT
				ad_tag_publisher_id,
				targeting_id,
				sum(requests) as requests,
				domain
			FROM statistics.statistics_merged
			WHERE date_time >= now() - 60*60*24 AND targeting_id IN (%s)
			GROUP BY ad_tag_publisher_id, targeting_id, domain
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_publisher_id,
				targeting_id,
				impressions,
				event_requests,
				domain
			FROM (
				SELECT
					ad_tag_publisher_id,
					targeting_id,
					sum(events_count) as event_requests,
					domain
				FROM statistics.statistics_events_merged
				WHERE date_time >= now() - 60*60*24 AND targeting_id IN (%s) AND event_name='request'
				GROUP BY ad_tag_publisher_id, targeting_id, domain
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_publisher_id,
					targeting_id,
					sum(events_count) as impressions,
					domain
				FROM statistics.statistics_events_merged
				WHERE date_time >= now() - 60*60*24 AND targeting_id IN (%s) AND event_name='impression'
				GROUP BY ad_tag_publisher_id, targeting_id, domain
			) USING (ad_tag_publisher_id, targeting_id, domain)
		) USING (ad_tag_publisher_id, targeting_id, domain)`, linksIDsJoined, linksIDsJoined, linksIDsJoined)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item ERPRDataByTargetingID
	var stats []ERPRDataByTargetingID
	for iter.Scan(
		&item.AdTagPublisherID, &item.TargetingID, &item.FillRate, &item.Domain,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return stats
}

func getPublisherLinksStatsByAdTag(publisherLinksIDs []string) []ERPRDataByTargetingID {
	var linksIDsString []string
	linksIDsString = make([]string, len(publisherLinksIDs))

	for i, id := range publisherLinksIDs {
		linksIDsString[i] = fmt.Sprintf("'%s'", id)
	}

	linksIDsJoined := strings.Join(linksIDsString, ",")

	queryString := fmt.Sprintf(`
		SELECT
			ad_tag_publisher_id,
			targeting_id,
			requests + event_requests,
			impressions,
			if (toUInt8(requests), (impressions / requests) * 100, 0) as fill_rate,
			if (toUInt8(impressions), (origin_amount - amount) / impressions * 1000, 0) as margin
		FROM (
			SELECT
				ad_tag_publisher_id,
				targeting_id,
				sum(requests) as requests
			FROM statistics.statistics_merged
			WHERE date_time >= now() - 60*60 AND targeting_id IN (%s)
			GROUP BY ad_tag_publisher_id, targeting_id
		)
		ANY FULL OUTER JOIN (
			SELECT
				ad_tag_publisher_id,
				targeting_id,
				impressions,
				amount,
				origin_amount,
				profit,
				event_requests
			FROM (
				SELECT
					ad_tag_publisher_id,
					targeting_id,
					sum(events_count) as event_requests
				FROM statistics.statistics_events_merged
				WHERE date_time >= now() - 60*60 AND targeting_id IN (%s) AND event_name='request'
				GROUP BY ad_tag_publisher_id, targeting_id
			)
			ANY FULL OUTER JOIN (
				SELECT
					ad_tag_publisher_id,
					targeting_id,
					sum(events_count) as impressions,
					sum(amount) / 1000000 as amount,
					sum(origin_amount) / 1000000 as origin_amount,
					(origin_amount - amount) as profit
				FROM statistics.statistics_events_merged
				WHERE date_time >= now() - 60*60 AND targeting_id IN (%s) AND event_name='impression'
				GROUP BY ad_tag_publisher_id, targeting_id
			) USING (ad_tag_publisher_id, targeting_id)
		) USING (ad_tag_publisher_id, targeting_id)`, linksIDsJoined, linksIDsJoined, linksIDsJoined)

	query := clickhouse.NewQuery(queryString)
	iter := query.Iter(database.ClickHouse)

	var item ERPRDataByTargetingID
	var stats []ERPRDataByTargetingID
	for iter.Scan(
		&item.AdTagPublisherID, &item.TargetingID, &item.Requests, &item.Impressions, &item.FillRate, &item.Margin,
	) {
		stats = append(stats, item)
	}

	if iter.Error() != nil {
		log.Warn(iter.Error())
	}

	return stats
}
