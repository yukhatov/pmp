$(function() {
    var datePickerInput = $('.multiple__input input');

    function showClock() {
        var d = new Date();
        var timezone = $("#clock").attr("data-timezone");
        document.getElementById("clock").innerHTML = d.toLocaleTimeString("en-US", {timeZone: timezone});
    }

    var interval = setInterval(function() {
        showClock();
    }, 1000);

    showClock();

    $('.search__ico').click(function (e) {
        e.preventDefault();
        submitSearch();
        return false;
    });

    $("#csv_export").click(function (e) {
        e.preventDefault();
        var start = $(datePickerInput).data('daterangepicker').startDate;
        var end = $(datePickerInput).data('daterangepicker').endDate;

        var url = '/publisher_admin/csv_export/?search=' +  $('#search_tags').val() + '&start_date=' + start.format('YYYY-MM-DD') + '&end_date=' + end.format('YYYY-MM-DD');
        window.location.href = url;
    });

    function submitSearch(start, end) {
        if (!start) {
            start = $(datePickerInput).data('daterangepicker').startDate;
        }
        if (!end) {
            end = $(datePickerInput).data('daterangepicker').endDate;
        }
        var url = '/publisher_admin/?search=' +  $('#search_tags').val() + '&start_date=' + start.format('YYYY-MM-DD') + '&end_date=' + end.format('YYYY-MM-DD');
        window.location.href = url;
    }

    var options = {
        locale: {
            format: 'YYYY-MM-DD'
        },
        ranges: {
            'Today': [moment(), moment()],
            'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
            'Last 7 Days': [moment().subtract(6, 'days'), moment()],
            'Last 30 Days': [moment().subtract(29, 'days'), moment()],
            'This Month': [moment().startOf('month'), moment().endOf('month')],
        },
        parentEl: datePickerInput.parent(),
        singleDatePicker: false
    };

    $(datePickerInput).daterangepicker(options, function(start, end, label) {
        submitSearch(start, end);
    });
});
