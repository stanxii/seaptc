//const server = "https://seaptc.org"
const server = "http://localhost:8080"

let tabURLs = [];

function fetchClass(request, sender, sendResponse) {
  // The session event page fetches class after page load.  Create next tab in
  // queue, if any.
  createNextTab();

  let url = `${server}/session-events/${request.num}`;
  console.log('Requesting', url);
  fetch(url)
    .then(response => response.json())
    .then(m => sendResponse(m));
}

function openTabs(request, sender, sendResponse) {
  tabURLs = request.urls;
  createNextTab();
}

function createNextTab() {
  let url = tabURLs.pop();
  if (url) {
    chrome.tabs.create({ url: url });
  }
}

let handlers = {
  "openTabs": openTabs,
  "fetchClass": fetchClass
};

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  let h = handlers[request.handler];
  if (!h) {
    console.log(`No handler for request ${request}`);
    return false;
  }
  h(request, sender, sendResponse);
  return true;
});
