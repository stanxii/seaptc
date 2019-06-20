"use strict";

function catchEm(promise) {
  return promise.then(value => [null, value], reason => [reason]);
}

function getTrap(target, property) {
  return function(...args) {
    return new Promise(function(resolve, reject) {
      target[property](...args, function(value) {
        if (chrome.runtime.lastError) {
          reject(chrome.runtime.lastError);
        } else {
          resolve(value);
        }
      })
    })
  }
}

function wrapext(path) {
  let target = chrome;
  for (name of path.split(".")) {
    target = target[name];
    if (target === undefined) {
      return undefined;
    }
  }
  return new Proxy(target, {get: getTrap});
}

const chromeTabs = wrapext("tabs");
const chromeRuntime = wrapext("runtime");
const chromeStorageSync = wrapext("storage.sync");

function callBackground(...nameAndArgs) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage(nameAndArgs, response => {
      if (chrome.runtime.lastError) {
        reject(chrome.runtime.lastError)
      } else {
        let [reason, value] = response;
        if (reason !== null) {
          reject(reason);
        } else {
          resolve(value);
        }
      }
    });
  });
}

function listenBackground(handlers) {
  chrome.runtime.onMessage.addListener(([name, ...args], sender, sendResponse) => {
    const handler = handlers[name];
    if (!handler) {
      sendResponse([`No handler for ${name}`]);
      return false;
    }
    Promise.resolve(handler(sender, ...args)).then(
      value => sendResponse([null, value]),
      reason => {
        if (reason instanceof Error && reason.message) {
          console.log(reason);
          reason = reason.message;
        }
        sendResponse([reason])
      });
    return true;
  });
}
