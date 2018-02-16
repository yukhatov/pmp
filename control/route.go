package control

import (
	"bitbucket.org/tapgerine/pmp/control/admin"

	"bitbucket.org/tapgerine/pmp/control/data_sync"

	"bitbucket.org/tapgerine/pmp/control/auth"
	"bitbucket.org/tapgerine/pmp/control/publisher_admin"
	"bitbucket.org/tapgerine/pmp/control/statistics_merge"
	"goji.io"
	"goji.io/pat"
)

func CreateRouter() *goji.Mux {
	router := goji.NewMux()

	router.Use(authMiddleware)
	router.HandleFunc(NewNamedPattern("ActionLogList", pat.Get("/action_log/")), admin.ActionLogListHandler)

	router.HandleFunc(NewNamedPattern("PlatformTypeList", pat.Get("/platform_type/list/")), admin.GetPlatformTypeListHandler)
	router.HandleFunc(NewNamedPattern("PlatformTypeCreate", pat.Get("/platform_type/create/")), admin.GetPlatformTypeCreateHandler)
	router.HandleFunc(NewNamedPattern("PlatformTypeEdit", pat.Get("/platform_type/:platform_type_id/edit/")), admin.GetPlatformTypeEditHandler)
	router.HandleFunc(NewNamedPattern("PlatformTypePostCreate", pat.Post("/platform_type/create/")), admin.PostPlatformTypeCreateHandler)
	router.HandleFunc(NewNamedPattern("PlatformTypePostEdit", pat.Post("/platform_type/:platform_type_id/edit/")), admin.PostPlatformTypeEditHandler)

	router.HandleFunc(NewNamedPattern("ParameterList", pat.Get("/parameter/list/")), admin.GetParamaterListHandler)
	router.HandleFunc(NewNamedPattern("ParameterCreate", pat.Get("/parameter/create/")), admin.GetParameterCreateHandler)
	router.HandleFunc(NewNamedPattern("ParameterEdit", pat.Get("/parameter/:parameter_id/edit/")), admin.GetParameterEditHandler)
	router.HandleFunc(NewNamedPattern("ParameterPostCreate", pat.Post("/parameter/create/")), admin.PostParameterCreateHandler)
	router.HandleFunc(NewNamedPattern("ParameterPostEdit", pat.Post("/parameter/:parameter_id/edit/")), admin.PostParameterEditHandler)
	router.HandleFunc(NewNamedPattern("ParameterPostDelete", pat.Post("/parameter/:parameter_id/delete/")), admin.PostPrameterDeleteHandler)

	router.HandleFunc(NewNamedPattern("ParameterMapList", pat.Get("/platform_type/:platform_type_id/parameter_map/list/")), admin.GetParamaterMapListHandler)
	router.HandleFunc(NewNamedPattern("ParameterMapCreate", pat.Get("/platform_type/:platform_type_id/parameter_map/create/")), admin.GetParameterMapCreateHandler)
	router.HandleFunc(NewNamedPattern("ParameterMapEdit", pat.Get("/platform_type/:platform_type_id/parameter_map/:parameter_map_id/edit/")), admin.GetParameterMapEditHandler)
	router.HandleFunc(NewNamedPattern("ParameterMapPostCreate", pat.Post("/platform_type/:platform_type_id/parameter_map/create/")), admin.PostParameterMapCreateHandler)
	router.HandleFunc(NewNamedPattern("ParameterMapPostEdit", pat.Post("/platform_type/:platform_type_id/parameter_map/:parameter_map_id/edit/")), admin.PostParameterMapEditHandler)

	router.HandleFunc(NewNamedPattern("AdvertiserList", pat.Get("/advertiser/list/")), admin.AdvertiserListHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserListJson", pat.Get("/advertiser/list/json/")), admin.AdvertiserListJson)
	router.HandleFunc(NewNamedPattern("AdvertiserListJsonByAdTag", pat.Get("/advertiser/by_ad_tag/list/:ad_tag_id/json/")), admin.AdvertiserListJsonByAdTag)
	router.HandleFunc(NewNamedPattern("AdvertiserCreate", pat.Get("/advertiser/create/")), admin.GetAdvertiserCreateHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserEdit", pat.Get("/advertiser/:advertiser_id/edit/")), admin.GetAdvertiserEditHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserEditPost", pat.Post("/advertiser/:advertiser_id/edit/")), admin.PostAdvertiserHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserListJson", pat.Get("/advertiser/by_publisher/list/:publisher_id/json/")), admin.AdvertiserListJsonByPublisher)

	router.HandleFunc(NewNamedPattern("PublisherList", pat.Get("/publisher/list/")), admin.PublisherListHandler)
	router.HandleFunc(NewNamedPattern("PublisherListJson", pat.Get("/publisher/list/json/")), admin.PublisherListJson)
	router.HandleFunc(NewNamedPattern("PublisherCreate", pat.Get("/publisher/create/")), admin.GetPublisherCreateHandler)
	router.HandleFunc(NewNamedPattern("PublisherEdit", pat.Get("/publisher/:publisher_id/edit/")), admin.GetPublisherEditHandler)

	router.HandleFunc(NewNamedPattern("GetAdTagPublisherLinkAdd", pat.Get("/ad_tag/:ad_tag_id/publisher_link/list/")), admin.AdTagPublisherLinkListHandler)
	router.HandleFunc(NewNamedPattern("PostAdTagPublisherLinkConnect", pat.Post("/ad_tag/:ad_tag_id/:publisher_link_id/connect/")), admin.AdTagPublisherLinkConnect)
	router.HandleFunc(NewNamedPattern("PostAdTagPublisherLinkDisconnect", pat.Post("/ad_tag/:ad_tag_id/:publisher_link_id/disconnect/")), admin.AdTagPublisherLinkDisconnect)

	router.HandleFunc(NewNamedPattern("PublisherLinksListJson", pat.Get("/publisher/:publisher_id/link/list/json/")), admin.PublisherLinksListJson)
	router.HandleFunc(NewNamedPattern("PublisherLinksList", pat.Get("/publisher/:publisher_id/link/list/")), admin.PublisherLinksListHandler)
	router.HandleFunc(NewNamedPattern("GetPublisherLinkCreate", pat.Get("/publisher/:publisher_id/link/create/")), admin.GetPublisherLinkCreateHandler)
	router.HandleFunc(NewNamedPattern("PostPublisherLinkCreate", pat.Post("/publisher/:publisher_id/link/create/")), admin.PostPublisherLinkCreateHandler)

	router.HandleFunc(NewNamedPattern("GetPublisherLinkEdit", pat.Get("/publisher/:publisher_id/link/:link_id/edit/")), admin.GetPublisherLinkEditHandler)
	router.HandleFunc(NewNamedPattern("PostPublisherLinkEdit", pat.Post("/publisher/:publisher_id/link/:link_id/edit/")), admin.PostPublisherLinkEditHandler)

	router.HandleFunc(NewNamedPattern("PublisherLinksAdTagPublisherList", pat.Get("/publisher/:publisher_id/link/:link_id/list/")), admin.PublisherLinksAdTagPublisherListHandler)
	router.HandleFunc(NewNamedPattern("GetPublisherLinkAdTagPublisherCreate", pat.Get("/publisher/:publisher_id/link/:link_id/add_ad_tag_publisher/")), admin.GetPublisherLinkAdTagPublisherCreateHandler)
	router.HandleFunc(NewNamedPattern("PostPublisherLinkAdTagPublisherCreate", pat.Post("/publisher/:publisher_id/link/:link_id/add_ad_tag_publisher/")), admin.PostPublisherLinkAdTagPublisherCreateHandler)

	router.HandleFunc(NewNamedPattern("GetPublisherLinkAdTagCreate", pat.Get("/publisher/:publisher_id/link/:link_id/add_tag/")), admin.GetPublisherLinkAdTagCreateHandler)
	router.HandleFunc(NewNamedPattern("PostPublisherLinkAdTagCreate", pat.Post("/publisher/:publisher_id/link/:link_id/add_tag/")), admin.PostPublisherLinkAdTagCreateHandler)

	router.HandleFunc(NewNamedPattern("PublisherLinkAdTagActivationHandler", pat.Post("/publisher_link_ad_tag_publisher/:id/activation/")), admin.PublisherLinkAdTagActivationHandler)

	router.HandleFunc(NewNamedPattern("PublisherEditPost", pat.Post("/publisher/:publisher_id/edit/")), admin.PostPublisherHandler)
	router.HandleFunc(NewNamedPattern("PublisherAdTagsList", pat.Get("/publisher/:publisher_id/ad_tag/list/")), admin.PublisherAdTagsListHandler)
	router.HandleFunc(NewNamedPattern("PublisherUserEdit", pat.Get("/publisher/:publisher_id/user/edit/")), admin.PublisherUserEditHandler)
	router.HandleFunc(NewNamedPattern("PublisherUserEditPost", pat.Post("/publisher/:publisher_id/user/edit/")), admin.PublisherUserEditHandler)
	router.HandleFunc(NewNamedPattern("LogAsPublisher", pat.Get("/publisher/:publisher_id/log_as_pub/")), admin.LogAsPublisherHandler)
	router.HandleFunc(NewNamedPattern("PublisherListByAdvertJson", pat.Get("/publisher/list/:advertiser_id/json/")), admin.PublisherListJsonByAdvertiser)
	router.HandleFunc(NewNamedPattern("PublisherListByAdTagJson", pat.Get("/publisher/by_ad_tag/list/:ad_tag_id/json/")), admin.PublisherListJsonByAdTag)
	router.HandleFunc(NewNamedPattern("PublisherAdTagCreate", pat.Get("/publisher/:publisher_id/ad_tag/create/")), admin.GetPublisherAdTagCreateHandler)
	router.HandleFunc(NewNamedPattern("PublisherAdTagCreatePost", pat.Post("/publisher/:publisher_id/ad_tag/create/")), admin.PostPublisherAdTagCreateHandler)

	router.HandleFunc(NewNamedPattern("AdTagListJsonByAdvertAndPub", pat.Get("/ad_tag/list/:advertiser_id/:publisher_id/json/")), admin.AdTagsListJsonByAdvertAndPub)
	router.HandleFunc(NewNamedPattern("AdTagListJsonByAdvert", pat.Get("/ad_tag/by_advertiser/list/:advertiser_id/json/")), admin.AdTagsListJsonByAdvertiser)
	router.HandleFunc(NewNamedPattern("AdTagListJsonByPub", pat.Get("/ad_tag/by_publisher/list/:publisher_id/json/")), admin.AdTagsListJsonByPublisher)
	router.HandleFunc(NewNamedPattern("AdTagListJson", pat.Get("/ad_tag/list/json/")), admin.AdTagsListJson)
	router.HandleFunc(NewNamedPattern("AdTagList", pat.Get("/ad_tag/list/:advertiser_id")), admin.AdTagsListHandler)
	router.HandleFunc(NewNamedPattern("AdTagCreate", pat.Get("/ad_tag/create/:advertiser_id")), admin.AdTagCreateHandler)
	router.HandleFunc(NewNamedPattern("AdTagEdit", pat.Get("/ad_tag/:ad_tag_id/edit/")), admin.AdTagEditHandler)
	router.HandleFunc(NewNamedPattern("AdTagEditPost", pat.Post("/ad_tag/:ad_tag_id/edit/")), admin.AdTagEditHandler)
	router.HandleFunc(NewNamedPattern("AdTagStatistics", pat.Get("/ad_tag/:ad_tag_id/statistics/")), admin.AdTagStatisticsHandler)
	router.HandleFunc(NewNamedPattern("AdTagActivation", pat.Post("/ad_tag/:ad_tag_id/activation/")), admin.AdTagActivationHandler)
	router.HandleFunc(NewNamedPattern("AdTagArchive", pat.Post("/ad_tag/:ad_tag_id/archive/")), admin.AdTagArchiveHandler)
	router.HandleFunc(NewNamedPattern("FindDeadTags", pat.Get("/ad_tag/find_dead_tags/")), admin.FindDeadTagsHandler)

	router.HandleFunc(NewNamedPattern("GetPublisherForAdTag", pat.Get("/ad_tag/:ad_tag_id/publisher/list/json/")), admin.GetPublisherForAdTagHandler)

	router.HandleFunc(NewNamedPattern("AdTagPublisherList", pat.Get("/ad_tag/:ad_tag_id/publisher/list/")), admin.AdTagPublisherListHandler)
	router.HandleFunc(NewNamedPattern("AdTagPublisherAdd", pat.Get("/ad_tag/:ad_tag_id/publisher/add/")), admin.AdTagPublisherAddHandler)
	router.HandleFunc(NewNamedPattern("AdTagPublisherEdit", pat.Get("/ad_tag/:ad_tag_id/publisher/edit/:id")), admin.AdTagPublisherEditHandler)
	router.HandleFunc(NewNamedPattern("AdTagPublisherEditPost", pat.Post("/ad_tag/:ad_tag_id/publisher/edit/:id")), admin.AdTagPublisherEditHandler)
	router.HandleFunc(NewNamedPattern("AdTagPublisherActivationPost", pat.Post("/ad_tag/:ad_tag_id/publisher/activation/:id")), admin.AdTagPublisherActivationHandler)

	router.HandleFunc(NewNamedPattern("Statistics", pat.Get("/statistics")), admin.StatisticsMainHandler)
	router.HandleFunc(NewNamedPattern("StatisticsRtb", pat.Get("/statistics_rtb")), admin.RtbStatisticsMainHandler)

	router.HandleFunc(NewNamedPattern("Sync", pat.Get("/sync")), dataSync.SyncDataIntoRedis)

	router.HandleFunc(NewNamedPattern("MergeClickHouseStatistics", pat.Get("/merge_statistics")), statisticsMerge.MergeClickHouseRequestStatistics)
	router.HandleFunc(NewNamedPattern("MergeClickHouseEventsStatistics", pat.Get("/merge_statistics_events")), statisticsMerge.MergeClickHouseEventsStatistics)
	router.HandleFunc(NewNamedPattern("MergeClickHouseStatistics", pat.Get("/merge_statistics_rtb_events")), statisticsMerge.MergeClickHouseRtbEventsStatistics)
	router.HandleFunc(NewNamedPattern("MergeClickHouseStatistics", pat.Get("/merge_statistics_rtb_requests")), statisticsMerge.MergeClickHouseRtbRequestsStatistics)

	router.HandleFunc(NewNamedPattern("Login", pat.Get("/login")), auth.LoginHandler)
	router.HandleFunc(NewNamedPattern("LoginPost", pat.Post("/login")), auth.LoginHandler)
	router.HandleFunc(NewNamedPattern("JWTTokenCheck", pat.Get("/jwt_token_check")), auth.JWTCheckTokenHandler)
	router.HandleFunc(NewNamedPattern("JWTLogin", pat.Get("/jwt_login")), auth.JWTLoginHandler)
	router.HandleFunc(NewNamedPattern("Logout", pat.Get("/logout")), auth.LogoutHandler)

	router.HandleFunc(NewNamedPattern("AdminUserCreate", pat.Get("/admin/user/create/")), admin.AdminUserCreateHandler)
	router.HandleFunc(NewNamedPattern("AdminUserEdit", pat.Get("/admin/user/:user_id/edit/")), admin.AdminUserEditHandler)
	router.HandleFunc(NewNamedPattern("AdminUserEditPost", pat.Post("/admin/user/:user_id/edit/")), admin.AdminUserEditHandler)

	router.HandleFunc(NewNamedPattern("MainPage", pat.Get("/")), admin.MainPageHandler)

	router.HandleFunc(NewNamedPattern("PublisherAdminMainPage", pat.Get("/publisher_admin/")), publisherAdmin.MainPageHandler)
	router.HandleFunc(NewNamedPattern("PublisherStatsCSVExport", pat.Get("/publisher_admin/csv_export/")), publisherAdmin.PublisherStatsCSVExportHandler)

	router.HandleFunc(NewNamedPattern("GeoMap", pat.Get("/geo_map")), admin.GeoMapIndex)
	router.HandleFunc(NewNamedPattern("GeoMap", pat.Get("/geo_map/data")), admin.GeoMapData)

	router.HandleFunc(NewNamedPattern("AdTagRecommendationList", pat.Get("/ad_tags_recommendation/")), admin.AdTagRecommendationListHandler)
	router.HandleFunc(NewNamedPattern("AdTagRecommendationProcessor", pat.Get("/ad_tags_recommendation/processor/")), admin.AdTagRecommendationProcessorHandler)
	router.HandleFunc(NewNamedPattern("AdTagRecommendationFixed", pat.Post("/ad_tags_recommendation/:id/fixed/")), admin.AdTagRecommendationFixedHandler)

	router.HandleFunc(NewNamedPattern("DomainsList", pat.Get("/domains/")), admin.DomainsListHandler)
	router.HandleFunc(NewNamedPattern("DomainsListCreate", pat.Get("/domains/create/")), admin.DomainsListCreateHandler)
	router.HandleFunc(NewNamedPattern("DomainsListEdit", pat.Get("/domains/:domains_list_id/edit/")), admin.DomainsListEditHandler)
	router.HandleFunc(NewNamedPattern("DomainsListEditPost", pat.Post("/domains/:domains_list_id/edit/")), admin.DomainsListEditHandler)

	router.HandleFunc(NewNamedPattern("PublisherInvoiceList", pat.Get("/invoice/publisher/list/")), admin.PublisherInvoiceListHandler)
	router.HandleFunc(NewNamedPattern("PublisherInvoiceCreate", pat.Get("/invoice/publisher/create/")), admin.PublisherInvoiceCreateHandler)
	router.HandleFunc(NewNamedPattern("PublisherInvoiceEdit", pat.Get("/invoice/publisher/:id/edit/")), admin.PublisherInvoiceEditHandler)
	router.HandleFunc(NewNamedPattern("PublisherInvoiceEditPost", pat.Post("/invoice/publisher/:id/edit/")), admin.PublisherInvoiceEditHandler)
	router.HandleFunc(NewNamedPattern("PublisherInvoiceStatusChange", pat.Post("/invoice/publisher/:id/status_change/:status")), admin.PublisherInvoiceStatusChangeHandler)
	router.HandleFunc(NewNamedPattern("PublisherInvoiceView", pat.Get("/invoice/publisher/:id/view_invoice/")), admin.PublisherInvoiceViewHandler)
	router.HandleFunc(NewNamedPattern("PublisherInvoiceDetails", pat.Get("/invoice/publisher/:id/details/")), admin.PublisherInvoiceDetailsHandler)

	router.HandleFunc(NewNamedPattern("AdvertiserInvoiceList", pat.Get("/invoice/advertiser/list/")), admin.AdvertiserInvoiceListHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserInvoiceCreate", pat.Get("/invoice/advertiser/create/")), admin.AdvertiserInvoiceCreateHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserInvoiceEdit", pat.Get("/invoice/advertiser/:id/edit/")), admin.AdvertiserInvoiceEditHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserInvoiceEditPost", pat.Post("/invoice/advertiser/:id/edit/")), admin.AdvertiserInvoiceEditHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserInvoiceCreate", pat.Post("/invoice/advertiser/:id/status_change/:status")), admin.AdvertiserInvoiceStatusChangeHandler)

	router.HandleFunc(NewNamedPattern("AdvertiserGenerateInvoice", pat.Get("/invoice/advertiser/:id/generate_invoice/")), admin.AdvertiserGenerateInvoiceHandler)
	router.HandleFunc(NewNamedPattern("AdvertiserInvoiceView", pat.Get("/invoice/advertiser/:id/view_invoice/")), admin.AdvertiserInvoiceViewHandler)

	//// Register pprof handlers
	//router.HandleFunc(NewNamedPattern("/debug/pprof/", pat.Get("/debug/pprof/")), pprof.Index)
	//router.HandleFunc(NewNamedPattern("/debug/pprof/cmdline", pat.Get("/debug/pprof/cmdline")), pprof.Cmdline)
	//router.HandleFunc(NewNamedPattern("/debug/pprof/profile", pat.Get("/debug/pprof/profile")), pprof.Profile)
	//router.HandleFunc(NewNamedPattern("/debug/pprof/symbol", pat.Get("/debug/pprof/symbol")), pprof.Symbol)
	//router.HandleFunc(NewNamedPattern("/debug/pprof/trace", pat.Get("/debug/pprof/trace")), pprof.Trace)

	return router
}
