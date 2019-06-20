"use strict";

const server = "https://seaptc.org";
//const server = "http://localhost:8080";

let sessionEventURLs = [];

async function createNextSessionEventTab() {
  let url = sessionEventURLs.pop();
  if (url) {
    await chromeTabs.create({ url: url });
  }
}

async function createSessionEventTabs(sender, urls) {
  sessionEventURLs = urls;
  await createNextSessionEventTab();
}

async function fetchClass(sender, number) {
  await createNextSessionEventTab();
  let url = `${server}/api/sessionEvents/${number}`;
  let response = await fetch(url);
  let m = await response.json();
  if (m.error) {
    throw m.error;
  }
  return m.result;
}

/*
async function uploadExportFile(request, sender, sendResponse) {
  let response = await fetch(request.file);
  let blob = await response.blob();

  response = await fetch(request.server + "/api/uploadToken");
  let token = await response.json();

  let formData = new FormData();
  formData.append(token.result.name, token.result.value);
  formData.append("file", blob);

  response = await fetch(request.server + "/api/uploadRegistrations", {
    method: "POST",
    body: formData
  });

  let x = await response.json();
  sendResponse(x);
}
*/

listenBackground({
  "createSessionEventTabs": createSessionEventTabs,
  "fetchClass": fetchClass
});
