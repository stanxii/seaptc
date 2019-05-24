(function() {
  let desc = document.getElementsByName("Description")[0].value;
  let num = desc.substr(0, 3);
  fetch('https://seaptc.org/dashboard/sessionevents/' + num)
  .then(response => response.json())
  .then(cls => {
    console.log('---');
    let updates = '';
    let missing = '';
    for (let key in cls) {
      if (key == "Notes") {
        CKEDITOR.instances["Notes"].setData(cls[key]);
      } else {
        let x = document.getElementsByName(key)[0];
        if (!x || x.type == "hidden") {
          missing = missing + ' ' + key;
        }
        if (x.value != cls[key]) {
          updates = updates + '\n' + key + ': ' + x.value + ' -> ' + cls[key];
          x.value = cls[key];
        }
      }
    }
    if (missing) {
      alert('Could not find fields' + missing + '\n\nUpdates:' + updates);
      return;
    }
    if (updates && confirm('Update class ' + num + '?\n' + updates)) {
      setTimeout(function() { document.getElementById('iSaveEditEventButton').click(); }, 50);
    }
  });
})();
