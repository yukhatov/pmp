$( document ).ready(function() {
    $(document).on("click", "#parameter-remove", function(event){
        var id = $(this).attr("data-parameter-id");

        $.post("/parameter/" + id + '/delete/');

        event.preventDefault();
        location.reload();
    });
});