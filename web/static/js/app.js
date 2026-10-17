// Shows a message when an HTMX request fails (htmx ignores 4xx/5xx responses by default).
document.body.addEventListener("htmx:responseError", function (evt) {
  var flash = document.getElementById("flash");
  if (!flash) { return; }
  var messages = {
    403: "You are not allowed to do this.",
    404: "This book no longer exists.",
    409: "This book has loan history and cannot be deleted."
  };
  flash.textContent = messages[evt.detail.xhr.status] || "Request failed. Please try again.";
  flash.hidden = false;
});
document.body.addEventListener("htmx:beforeRequest", function () {
  var flash = document.getElementById("flash");
  if (flash) { flash.hidden = true; flash.textContent = ""; }
});
