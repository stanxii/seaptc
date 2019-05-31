const server = "https://seaptc.org"
//const server = "http://localhost:8080"

function fetchClass(request, sender, sendResponse) {
  let url = `${server}/session-events/${request.num}`;
  console.log('Requesting', url);
  fetch(url)
    .then(response => response.json())
    .then(m => sendResponse(m));
}

let handlers = { "fetchClass": fetchClass };

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  let h = handlers[request.handler];
  if (!h) {
    console.log(`No handler for request ${request}`);
    return false;
  }
  h(request, sender, sendResponse);
  return true;
});
