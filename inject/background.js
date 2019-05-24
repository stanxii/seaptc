(function() {
    var inject = function(tab) {
	chrome.tabs.executeScript(tab.ib, {
                code: "var scr = document.createElement('script');" +
                    "scr.type='text/javascript';" + 
                    "scr.src=chrome.extension.getURL('inject.js');" +
                    "document.getElementsByTagName('head')[0].appendChild(scr);"
	});
    };
    chrome.browserAction.onClicked.addListener(inject);
})();
