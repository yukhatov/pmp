$( document ).ready(function() {
    onCheckbox();
});

function onCheckbox() {
    $(".connect").change(function() {
        if (this.checked) {
            request("/ad_tag/" + $(this).attr("data-ad-tag-id") + "/" + $(this).attr("data-publisher-link-id") + "/connect/");
        } else {
            request("/ad_tag/" + $(this).attr("data-ad-tag-id") + "/" + $(this).attr("data-publisher-link-id") + "/disconnect/");
        }
    });
}

function request(url) {
    $("#loader").show();

    $.ajax({
        type: "POST",
        url: url,
        data: null,
        async: true,
        success: function (response) {
            console.log('SUCCESS');
        },
        complete: function() {
            $("#loader").hide();
        }
    });
}