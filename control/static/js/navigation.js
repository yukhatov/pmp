function goBack() {
    window.history.back();
}

function getStatisticDefaultParams() {
    window.location.href = "/statistics?start_date=" + moment().format('YYYY-MM-DD') + "&end_date=" + moment().format('YYYY-MM-DD');
}

function getStatisticRTBDefaultParams() {
    window.location.href = "/statistics_rtb?start_date=" + moment().format('YYYY-MM-DD') + "&end_date=" + moment().format('YYYY-MM-DD');
}

$( document ).ready(function() {
    $('input#yesterday').click(function(){
        var url = new URL(window.location.href);

        url.searchParams.set("start_date", moment().add(-1, 'days').format('YYYY-MM-DD'));
        url.searchParams.set("end_date", moment().add(-1, 'days').format('YYYY-MM-DD'));

        window.location.href = url;
    });

    $('input#today').click(function(){
        var url = new URL(window.location.href);

        url.searchParams.set("start_date", moment().format('YYYY-MM-DD'));
        url.searchParams.set("end_date", moment().format('YYYY-MM-DD'));

        window.location.href = url;
    });

    $('input#month').click(function(){
        var url = new URL(window.location.href);

        url.searchParams.set("start_date", moment().startOf('month').format('YYYY-MM-DD'));
        url.searchParams.set("end_date", moment().format('YYYY-MM-DD'));

        window.location.href = url;
    });

    $('input#csv').click(function(){
        var url = new URL(window.location.href);
        url.searchParams.set("csv_export", "true");
        window.location.href = url;
    });

    $('input#domains_export').click(function(){
        var url = new URL(window.location.href);
        url.searchParams.set("domains_export", "true");
        window.location.href = url;
    });

    $('#split a').click(function(){
        var url = new URL(window.location.href);

        url.searchParams.set("split_by", $(this).attr('id'));

        window.location.href = url;
    });

    $('a#ad_tag-filter').click(function(event){
        applyFilter({
            'group_by'   : "ad_tag",
            'advertiser' : 0,
            'publisher'  : 0,
            'ad_tag_publisher'  : 0,
            'ad_tag'     : $(this).attr('data-ad-tag-id'),
        });
    });

    $('a#advertiser-filter').click(function(event){
        applyFilter({
            'group_by'   : "ad_tag",
            'ad_tag'     : 0,
            'publisher'  : 0,
            'ad_tag_publisher'  : 0,
            'advertiser' : $(this).attr('data-advertiser-id'),
        });
    });

    $('a#publisher-filter').click(function(event){
        applyFilter({
            'group_by'   : "ad_tag",
            'ad_tag'     : 0,
            'advertiser' : 0,
            'ad_tag_publisher'  : 0,
            'pub_link'   : 0,
            'publisher'  : $(this).attr('data-publisher-id'),
        });
    });

    $('a#ad_tag_publisher-filter').click(function(event){
        applyFilter({
            'group_by'   : "ad_tag",
            'ad_tag'     : 0,
            'advertiser' : 0,
            'publisher'  : 0,
            'ad_tag_publisher'  : $(this).attr('data-ad-tag-publisher-id'),
        });
    });

    $('a#pub_link-filter').click(function(event){
        applyFilter({
            'group_by'   : "ad_tag",
            'ad_tag'     : 0,
            'advertiser' : 0,
            'publisher'  : $(this).attr('data-publisher-id'),
            'ad_tag_publisher'  : 0,
            'pub_link'   : $(this).attr('data-pub-link-id'),
        });
    });

    function applyFilter(params) {
        var url = new URL(window.location.href);

        for (var key in params) {
            url.searchParams.set(key, params[key]);
        }

        window.location.href = url;
    }
});