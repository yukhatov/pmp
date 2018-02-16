/**
 * Created by artur on 31.07.17.
 */
var form;
var advertiserId;

$(document).ready(function() {
    form = document.forms['ad-tag-publisher'];
    advertiserId = form.elements['advertiser'][form.elements['advertiser'].selectedIndex].value;

    filtersInit();
});

$(function() {
    filtersChange()
});

function filtersInit() {
    filtersUpdateData(advertiserId);
}

function filtersChange() {
    $("select#advertiser").change(function (e) {
        advertiserId = form.elements['advertiser'][form.elements['advertiser'].selectedIndex].value;

        filtersUpdateData(advertiserId);
    });
}

function filtersUpdateData(advertiserId) {
    setSelectOptions('select#ad_tag', getData("/ad_tag/by_advertiser/list/" + advertiserId + "/json/"));
}

function setSelectOptions(id, options) {
    $(id + ' option').remove();

    $(id).append($('<option>', {
        value: 0,
        text: "-"
    }));

    $.each(options, function (i, option) {
        $(id).append($('<option>', {
            value: option.ID,
            text: option.Name
        }));
    });

    $(id).selectpicker('refresh');
}

function getData(url) {
    var data = [];

    $.ajax({
        type: "GET",
        url: url,
        data: null,
        async: false,
        success: function (response) {
            data = response;
        }
    });

    return data;
}