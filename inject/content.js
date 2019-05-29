//const server = "https://seaptc.org"
const server = "http://localhost:8080"

var form = null;   // Look for controls in this form.
var info = null;   // Display information in this element.

// fetchClass fetches class with given number from the server and updates the
// contents of the info element as appropriate.
function fetchClass(num) {
  showMessage(`Fetching class ${num} from the server...`);
  fetch(`${server}/session-events/${num}`)
    .then(response => response.json())
    .then(m => {
      if (m.error) {
        showMessage(m.error);
      } else if (m.result) {
        showProposedChanges(controlValuesFromClass(m.result));
      } else {
        showMessage("Bad response from server.");
      }
    });
}

// showMessage replaces the info element contents with the given message and
// the class number button.
function showMessage(message) {
  while (info.firstChild) {
    info.removeChild(info.firstChild);
  }
  info.appendChild(createClassNumberButton());
  if (message) {
    info.appendChild(document.createElement("div")).textContent = message;
  }
}

// showProposedChanges replaces the info element content with button to save
// the changes and a definition list with the proposed changes (if any).
function showProposedChanges(values) {
  let dl = document.createElement("dl");
  let notFound = [];
  for (let [name, value] of values) {
    let control = form.querySelector(`[name="${name}"]`);
    if (!control || (name !== "Notes" && control.type === "hidden")) {
      notFound.push(name);
      continue;
    }
    if (control.value !== value) {
      dl.appendChild(document.createElement("dt")).textContent = `${name}:`;
      let dd = dl.appendChild(document.createElement("dd"));
      for (let v of [control.value, value]) {
        let d = dd.appendChild(document.createElement("div"))
        d.textContent = v ? v : "\u00A0";
        d.style = "white-space: nowrap; overflow: hidden; max-width: 500px;"
      }
    }
  }

  while (info.firstChild) {
    info.removeChild(info.firstChild);
  }
  info.appendChild(createSaveButton(values));
  info.appendChild(createClassNumberButton());

  if (notFound.length) {
    info.appendChild(document.createElement("div")).textContent = `Missing or hidden controls: ${notFound.join(", ")}`;
  }

  if (!dl.hasChildNodes()) {
    info.appendChild(document.createElement("div")).textContent = `No changes.`;
  } else {
    info.appendChild(dl);
  }
}

// setControlValues sets the given values on the form's controls.
function setControlValues(values) {
  for (let [name, value] of values) {
    if (name === "Notes") {
      let script = document.createElement("script")
      script.textContent = `CKEDITOR.instances["Notes"].setData(${JSON.stringify(value)})`
      document.head.appendChild(script);
      script.remove();
      continue
    }
    let control = form.querySelector(`[name="${name}"]`);
    if (!control || (control.type === "hidden")) {
      console.log(`DKE: control not found or hidden: ${name}}`)
      continue;
    }
    control.value = value;
  };
}

// createClassNumberButton creates a button with a click handler that prompts
// for and loads a class.
function createClassNumberButton() {
  let b = document.createElement("button");
  b.textContent = "Set class number";
  b.style = "margin: 5px;"
  b.onclick = (e) => {
    let num = window.prompt("Class number", getClassNumber());
    if (num) {
      fetchClass(num);
    }
    return false;
  };
  return b;
}

// createSaveButton creates a button that sets the given values on the controls
// and clicks the save button.
function createSaveButton(values) {
  let b = document.createElement("button");
  b.textContent = "Save";
  b.style = "margin: 5px;"
  b.onclick = (e) => {
    setControlValues(values);
    let b = form.querySelector("#iSaveEditEventButton");
    if (!b) {
      console.log("could not find save button");
      return false;
    }
    setTimeout(() => { b.click(); }, 50);
    return false;
  };
  return b;
}

// getClassNumber gets the class number from the description control or returns
// "" if the control is not found or empty.
function getClassNumber() {
  let description = form.querySelector("[name=\"Description\"]");
  if (!description) {
    console.log("DKE: could not find description control");
    return "";
  }
  let s = description.value;
  if (!s.match(/^\d\d\d:/)) {
    return "";
  }
  return s.substr(0, 3);
}

(() => {
  let title = document.querySelector("#pagetitlediv table tbody tr:first-of-type th");
  if (!title || (title.innerText.toUpperCase() !== "CREATE AND MODIFY A SESSION EVENT")) {
    console.log("DKE: could not find session event title on page");
    return;
  }

  form = document.querySelector("form[action=\"UpdateActivity.asp\"]");
  if (!form) {
    console.log("DKE: could not update activity form");
    return;
  }

  let tbody = form.querySelector("table > tbody");
  if (!tbody) {
    console.log("DKE: could not find form > table > tbody");
    return
  }

  let tr = tbody.insertBefore(document.createElement("tr"), tbody.firstChild);
  info = tr.appendChild(document.createElement("td"));
  info.colspan = 2;

  let num = getClassNumber();
  if (num) {
    fetchClass(num);
  } else {
    showMessage("");
  }

})();
