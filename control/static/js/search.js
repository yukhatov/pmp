var isPreloaderOn = 0;

$( document ).ready(function() {
    filtersInit();
});

$(function() {
    filtersChange();

    $('.sort-stats>span').click(function (e) {
        var field = $(this).parent().attr("data-field");
        var order = $(this).attr("data-order");
        var url = new URL(window.location.href);

        url.searchParams.set("order_by", field);
        url.searchParams.set("order_by_order", order);

        window.location.href = url;
    });
});

function validateForm(event) {
    var startDate = $("#start_date").val();
    var endDate = $("#end_date").val();

    if (startDate > endDate) {
        document.getElementById("validation_message").innerHTML = "<font color='red'>End date should be greater than start date</font>";
        document.getElementById("end_date").style.border = "#FF0000 1px solid";

        event.preventDefault();
    }
}

function filtersInit() {
    var url = new URL(window.location.href);

    var advertiserId = url.searchParams.get("advertiser") ? url.searchParams.get("advertiser") : 0;
    var publisherId = url.searchParams.get("publisher") ? url.searchParams.get("publisher") : 0;
    var adTagId = url.searchParams.get("ad_tag") ? url.searchParams.get("ad_tag") : 0;
    var groupById = url.searchParams.get("group_by") ? url.searchParams.get("group_by") : "ad_tag";
    var pubLinkId = url.searchParams.get("pub_link") ? url.searchParams.get("pub_link") : 0;

    filtersUpdateData(advertiserId, publisherId, adTagId, groupById, pubLinkId);
}

function filtersChange() {
    $("select").change(function (e) {
        var form = document.forms['filters'];

        var advertiserId = form.elements['advertiser'][form.elements['advertiser'].selectedIndex].value;
        var publisherId = form.elements['publisher'][form.elements['publisher'].selectedIndex].value;
        var adTagId = form.elements['ad_tag'][form.elements['ad_tag'].selectedIndex].value;
        var groupById = form.elements['group_by'][form.elements['group_by'].selectedIndex].value;
        var pubLinkId = form.elements['pub_link'][form.elements['pub_link'].selectedIndex].value;

        filtersUpdateData(advertiserId, publisherId, adTagId, groupById, pubLinkId);
    });
}

function setFilterDisabled(id, isDisabled) {
    var form = document.forms['filters'];

    form.elements[id].disabled = isDisabled;

    $(form.elements[id]).selectpicker('refresh');
}

function filtersUpdateData(advertiserId, publisherId, adTagId, groupById, pubLinkId) {
    setSelectOptions('select#group_by',
        [
            {ID:'ad_tag', Name:'By Ad Tags'},
            {ID:'advertiser', Name:'By Advertisers'},
            {ID:'publisher', Name:'By Publishers'},
            {ID:'ad_tag_publisher', Name:"By Publisher's URL"},
            {ID:'publisher_links', Name:"By Sources"},
            {ID:'publisher_links_with_ad_tags', Name:"By Sources and Ad tags"}
        ], groupById, false);

    updateFilterData('select#group_by', null, groupById);
    setFilterDisabled('pub_link', true);
    setFilterDisabled('group_by', false);

    if(advertiserId == 0 && publisherId == 0 && adTagId == 0 && groupById == "ad_tag") {
        setFilterDisabled('ad_tag', false);
        setFilterDisabled('advertiser', false);
        setFilterDisabled('publisher', false);
    } else if (advertiserId == 0 && publisherId != 0 && adTagId == 0 && groupById != "ad_tag") {
        setFilterDisabled('pub_link', false);
        setFilterDisabled('group_by', false);
        setFilterDisabled('ad_tag', true);
        setFilterDisabled('advertiser', true);
        setSelectOptions('select#group_by', [{ID:'publisher_links', Name:'By Sources'}, {ID:'ad_tag', Name:'By Ad Tags'}], groupById, false)
    } else if (advertiserId == 0 && publisherId != 0 && adTagId == 0 && groupById == "ad_tag") {
        setFilterDisabled('pub_link', false);
        setFilterDisabled('group_by', false);
        setFilterDisabled('ad_tag', false);
        setFilterDisabled('advertiser', false);
        setSelectOptions('select#group_by', [{ID:'publisher_links', Name:'By Sources'}, {ID:'ad_tag', Name:'By Ad Tags'}], groupById, false)
    } else if (advertiserId == 0 && publisherId == 0 && adTagId == 0 && groupById != "ad_tag") {
        setFilterDisabled('ad_tag', true);
        setFilterDisabled('advertiser', true);
        setFilterDisabled('publisher', true);
    }

    if (advertiserId == 0 && publisherId == 0 && adTagId == 0) {
        updateFilterData('select#advertiser', "/advertiser/list/json/", advertiserId);
        updateFilterData('select#publisher', "/publisher/list/json/", publisherId);
        updateFilterData('select#ad_tag', "/ad_tag/list/json/", adTagId);
    }  else  if (advertiserId != 0 && publisherId != 0 && adTagId != 0){
        updateFilterData('select#ad_tag', "/ad_tag/list/" + advertiserId + "/" + publisherId + "/json/", adTagId);
        updateFilterData('select#advertiser', "/advertiser/by_ad_tag/list/"+ adTagId +"/json/", advertiserId);
        updateFilterData('select#publisher', "/publisher/by_ad_tag/list/"+ adTagId +"/json/", publisherId);
    } else if (advertiserId != 0 && publisherId == 0 && adTagId == 0) {
        updateFilterData('select#advertiser', "/advertiser/list/json/", advertiserId);
        updateFilterData('select#ad_tag', "/ad_tag/by_advertiser/list/" + advertiserId + "/json/");
        updateFilterData('select#publisher', "/publisher/list/" + advertiserId + "/json/");
    } else if (advertiserId != 0 && publisherId == 0 && adTagId != 0) {
         updateFilterData('select#publisher', "/publisher/by_ad_tag/list/"+ adTagId +"/json/");
         updateFilterData('select#ad_tag', "/ad_tag/by_advertiser/list/" + advertiserId + "/json/", adTagId);
         updateFilterData('select#advertiser', "/advertiser/list/json/", advertiserId);
    } else if (advertiserId != 0 && publisherId != 0 && adTagId == 0) {
         updateFilterData('select#ad_tag', "/ad_tag/list/" + advertiserId + "/" + publisherId + "/json/");
         updateFilterData('select#advertiser', "/advertiser/by_publisher/list/"+ publisherId +"/json/", advertiserId);
         updateFilterData('select#publisher', "/publisher/list/" + advertiserId + "/json/", publisherId);
    } else if (advertiserId == 0 && publisherId != 0 && adTagId == 0) {
        updateFilterData('select#advertiser', "/advertiser/by_publisher/list/" + publisherId + "/json/");
        updateFilterData('select#ad_tag', "/ad_tag/by_publisher/list/" + publisherId + "/json/");
        updateFilterData('select#publisher', "/publisher/list/json/", publisherId);
        updateFilterData('select#pub_link', "/publisher/" + publisherId + "/link/list/json/", pubLinkId);
    } else if (advertiserId == 0 && publisherId != 0 && adTagId != 0) {
        updateFilterData('select#publisher', "/publisher/by_ad_tag/list/"+ adTagId +"/json/", publisherId);
        updateFilterData('select#ad_tag', "/ad_tag/by_publisher/list/" + publisherId + "/json/", adTagId);
        updateFilterData('select#advertiser', "/advertiser/by_ad_tag/list/"+ adTagId +"/json/");
    } else if (advertiserId == 0 && publisherId == 0 && adTagId != 0) {
        updateFilterData('select#advertiser', "/advertiser/by_ad_tag/list/"+ adTagId +"/json/");
        updateFilterData('select#publisher', "/publisher/by_ad_tag/list/"+ adTagId +"/json/");
        updateFilterData('select#ad_tag', "/ad_tag/list/json/", adTagId);
    }
}

function updateFilterData(id, url, selectedId = 0) {
    if (url) {
        isPreloaderOn += 1;
        $("#loader").show();

        $.ajax({
            type: "GET",
            url: url,
            data: null,
            async: true,
            success: function (response) {
                setSelectOptions(id, response, selectedId);
            },
            complete: function() {
                isPreloaderOn -= 1;

                if (isPreloaderOn == 0) {
                    $("#loader").hide();
                }
            }
        });
    } else {
        setSelectOptions(id, null, selectedId)
    }
}

function setSelectOptions(id, options, selectedId, isDefaultOptionNeeded = true) {
    if (options) {
        $(id + ' option').remove();

        if (isDefaultOptionNeeded) {
            $(id).append($('<option>', {
                value: 0,
                text: "-"
            }));
        }

        $.each(options, function (i, option) {
            $(id).append($('<option>', {
                value: option.ID,
                text: option.Name
            }));
        });
    }

    $(id).selectpicker('val', selectedId);
    $(id).selectpicker('refresh');
}

function setFilterVisibility(filterName, isVisible) {
    var form = document.forms['filters'];

    form.elements[filterName].parentElement.style.visibility = !isVisible ? "hidden" : '';
}