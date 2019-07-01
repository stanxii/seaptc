"use strict";

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
  let settings = await chromeStorageSync.get(defaultSettings);
  let url = new URL(`/api/sessionEvents/${number}`, settings.server);
  let response = await fetch(url);
  let m = await response.json();
  if (m.error) {
    throw m.error;
  }
  return m.result;
}

async function uploadExportFile(sender) {
  let settings = await chromeStorageSync.get(defaultSettings);
  let response = await fetch(settings.exportPage, {
      cache: "no-cache",
      referrer: "no-referrer",
      redirect: "manual"
    });
  if (response.type === "opaqueredirect") {
    throw new Error(`Could net get export page: redirect, possibly to login page`);
  }
  let text = await response.text();
  let match = text.match(/\shref="(\/Handlers\/FileDownload.ashx\?FilePathName=[^"]*)"/);
  if (!match) {
    throw new Error(`Could not find download file on export page (bad setting for export page?)`);
  }
  let url = new URL(match[1], settings.exportPage);
  response = await fetch(url);
  let csv = await response.blob();

  url = new URL("/api/uploadRegistrationsToken", settings.server);
  response = await fetch(url);
  let token = await response.json();

  let formData = new FormData();
  formData.append(token.result.name, token.result.value);
  formData.append("file", csv);

  url = new URL("/api/uploadRegistrations", settings.server);
  response = await fetch(url, { method: "POST", body: formData });
  return await response.json();
}

listen({
  "createSessionEventTabs": createSessionEventTabs,
  "fetchClass": fetchClass,
  "uploadExportFile": uploadExportFile
});

chrome.browserAction.onClicked.addListener(
  () => chrome.tabs.create({ url: chrome.runtime.getURL("dashboard.html") }));

