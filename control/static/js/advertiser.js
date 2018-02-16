var RTBInputId = "rtb_integration_url";
var checkboxId = "is_dsp";

$( document ).ready(function() {
    setInputDisabled(RTBInputId, !document.getElementById(checkboxId).checked);
    onCheckbox();
});

function onCheckbox() {
    $("#is_dsp").change(function() {
        setInputDisabled(RTBInputId, !this.checked);
    });
}

function setInputDisabled(id, isDisabled) {
    var input = document.getElementById(id);

    input.disabled = isDisabled;
}