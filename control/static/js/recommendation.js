$(document).ready(function() {
    $('.activate_recommendation').click(function () {
        $("#loader").show();
        $.ajax({
            type: "POST",
            url: "/ad_tag/" + $(this).attr("data-id") + "/activation/",
            data: null,
            success: function () {
                $("#loader").hide();
            },
            error: function () {
                $("#loader").hide();
                //TODO: add normal error handling
                console.log("error while saving is_active");
            }
        })
    });

    $('.fixed').click(function () {
        $("#loader").show();
        var this_ = this;
        $.ajax({
            type: "POST",
            url: "/ad_tags_recommendation/" + $(this).attr("data-id") + "/fixed/",
            data: null,
            success: function () {
                $(this_).parents('tr').remove();
                $("#loader").hide();
            },
            error: function () {
                $("#loader").hide();
                //TODO: add normal error handling
                console.log("error while saving fixed");
            }
        })
    })
});