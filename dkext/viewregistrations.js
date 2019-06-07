// Content script export page.

(() => {
  let fileAnchor = document.querySelector("#lnkDownLoadFile")
  if (!fileAnchor) {
    console.log("DKE: could not link to file download");
    return;
  }

  let fileURL = new URL(fileAnchor.href, window.location.href).toString();
  let prodURL = new URL("https://seaptc.org/dashboard/fetch-registrations");
  prodURL.searchParams.set("url", fileURL);
  let devURL = new URL("http://localhost:8080/dashboard/fetch-registrations");
  devURL.searchParams.set("url", fileURL);

  let div = document.querySelector("#displayResults")
  if (!div) {
    console.log("DKE: could not display results");
    return;
  }

  let a = div.appendChild(document.createElement('a'));
  a.textContent = "Upload to seaptc.org";
  a.href = prodURL.toString();

  div.appendChild(document.createTextNode(" | "));

  a = div.appendChild(document.createElement('a'));
  a.textContent = "Upload to localhost:8080";
  a.href = devURL.toString();

})();
