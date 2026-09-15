// Evacuation roll-call behaviour.
//
// Loaded after the inline <script> in firelist.html, which declares the two
// globals this file reads:
//   fireListDay   — '2026-09-15', the day the server snapshot is for
//   fireListGrade — '5' | 'staff', the raw URL segment
// Do not redeclare them here.
//
// Everything in this file has to keep working after the network dies: the page
// is opened once at the start of an alarm and then used for minutes with no
// further requests. Nothing below talks to the server.

// ---- Storage ----
// One key per day and grade, so two teachers on two grades never share a list
// and yesterday's drill can never be mistaken for today's.
var FL_PREFIX = 'firelist:';
var flStorageKey = FL_PREFIX + fireListDay + ':' + fireListGrade;

// localStorage throws outright in Safari private mode and when a device is out
// of quota. Every access goes through these two so a storage failure costs the
// teacher their persistence, never the page.
function flLoadTicked() {
    try {
        var raw = window.localStorage.getItem(flStorageKey);
        if (!raw) return [];
        var parsed = JSON.parse(raw);
        return Object.prototype.toString.call(parsed) === '[object Array]' ? parsed : [];
    } catch (e) {
        return [];
    }
}

function flSaveTicked(ids) {
    try {
        window.localStorage.setItem(flStorageKey, JSON.stringify(ids));
    } catch (e) {
        // Nothing to do: the ticks stay correct on screen for this session.
    }
}

// Drop every firelist entry from an earlier day. Without this the next alarm
// would start with a partially ticked list restored from the previous one — a
// child shown as accounted for who has not been seen at all today.
function flPurgeOldDays() {
    try {
        var stale = [];
        var i;
        for (i = 0; i < window.localStorage.length; i++) {
            var key = window.localStorage.key(i);
            if (!key || key.indexOf(FL_PREFIX) !== 0) continue;
            // 'firelist:<day>:<grade>' — anything not for today goes.
            if (key.indexOf(FL_PREFIX + fireListDay + ':') !== 0) stale.push(key);
        }
        // Removal is a second pass: deleting inside the loop reindexes the keys.
        for (i = 0; i < stale.length; i++) {
            window.localStorage.removeItem(stale[i]);
        }
    } catch (e) {
        // Ignored on purpose.
    }
}

// ---- Elements ----
var flBody = document.getElementById('flBody');
var flDone = document.getElementById('flDone');
var flLeft = document.getElementById('flLeft');
var flResetBtn = document.getElementById('flReset');
var flReloadBtn = document.getElementById('flReload');

// The error and empty-list renderings have no table at all.
if (flBody) {
    flInit();
}

function flInit() {
    flPurgeOldDays();
    flRestore();
    flUpdateSummary();

    flBody.addEventListener('click', flOnRowClick);

    var heads = document.querySelectorAll('th[data-sort]');
    for (var i = 0; i < heads.length; i++) {
        heads[i].addEventListener('click', flOnSortClick);
        heads[i].addEventListener('keydown', flOnSortKey);
    }

    if (flResetBtn) flResetBtn.addEventListener('click', flOnReset);
    if (flReloadBtn) flReloadBtn.addEventListener('click', flOnReload);
}

// ---- Ticking ----

function flRows() {
    return flBody.getElementsByTagName('tr');
}

function flRestore() {
    var ticked = flLoadTicked();
    if (!ticked.length) return;

    var wanted = {};
    var i;
    for (i = 0; i < ticked.length; i++) {
        wanted[String(ticked[i])] = true;
    }

    var rows = flRows();
    for (i = 0; i < rows.length; i++) {
        if (!wanted[rows[i].getAttribute('data-id')]) continue;
        var box = rows[i].querySelector('.fl-check');
        if (box) box.checked = true;
        rows[i].className = 'is-checked';
    }
}

function flCollectTicked() {
    var rows = flRows();
    var ids = [];
    for (var i = 0; i < rows.length; i++) {
        if (rows[i].className.indexOf('is-checked') !== -1) {
            ids.push(rows[i].getAttribute('data-id'));
        }
    }
    return ids;
}

function flUpdateSummary() {
    if (!flDone || !flLeft) return;
    var total = flRows().length;
    var done = flCollectTicked().length;
    flDone.textContent = done;
    flLeft.textContent = total - done;
}

function flSetRow(row, checked) {
    var box = row.querySelector('.fl-check');
    if (box) box.checked = checked;
    row.className = checked ? 'is-checked' : '';
}

function flOnRowClick(event) {
    var target = event.target;
    var row = target;
    while (row && row.tagName !== 'TR') {
        row = row.parentNode;
    }
    if (!row || row.parentNode !== flBody) return;

    var box = row.querySelector('.fl-check');
    if (!box) return;

    // A tap that landed on the checkbox has already flipped it (and a keyboard
    // space press arrives here as a click too). Toggling again would undo the
    // teacher's tap, so in that case only the row class and the count are
    // brought in line with what the control already says.
    if (target !== box) {
        box.checked = !box.checked;
    }

    flSetRow(row, box.checked);
    flUpdateSummary();
    flSaveTicked(flCollectTicked());
}

// ---- Sorting ----
// Reorders only what the server already sent, in priority order. Ticks live on
// the row elements themselves, so moving the rows carries them along.

var flSortKey = '';
var flSortAsc = true;

// sensitivity 'base' makes Ivanov/ivanov/Ivanóv compare as equal-but-ordered
// rather than sorting all the accented names into a block of their own, which
// a plain < comparison on strings would do to this school's roll.
var flCollator = new Intl.Collator(undefined, { sensitivity: 'base' });

function flCompare(key, a, b) {
    if (key === 'status') {
        // data-rank is the server's priority order: 0 signed_in, 1 signed_out,
        // 2 not_signed. Sorting the label text instead would put "No Status"
        // between the two that matter.
        return parseInt(a.getAttribute('data-rank'), 10) - parseInt(b.getAttribute('data-rank'), 10);
    }
    return flCollator.compare(
        a.getAttribute('data-' + key) || '',
        b.getAttribute('data-' + key) || ''
    );
}

function flSortBy(key) {
    if (flSortKey === key) {
        flSortAsc = !flSortAsc;
    } else {
        flSortKey = key;
        flSortAsc = true;
    }

    var rows = [];
    var live = flRows();
    var i;
    for (i = 0; i < live.length; i++) {
        rows.push(live[i]);
    }

    var direction = flSortAsc ? 1 : -1;
    rows.sort(function(a, b) {
        return flCompare(key, a, b) * direction;
    });

    // One fragment, one insertion: reparenting 24 rows individually would make
    // the browser lay the table out 24 times.
    var frag = document.createDocumentFragment();
    for (i = 0; i < rows.length; i++) {
        frag.appendChild(rows[i]);
    }
    flBody.appendChild(frag);

    var heads = document.querySelectorAll('th[data-sort]');
    for (i = 0; i < heads.length; i++) {
        var active = heads[i].getAttribute('data-sort') === key;
        heads[i].setAttribute('aria-sort', active ? (flSortAsc ? 'ascending' : 'descending') : 'none');
    }
}

function flOnSortClick(event) {
    flSortBy(event.currentTarget.getAttribute('data-sort'));
}

function flOnSortKey(event) {
    if (event.key !== 'Enter' && event.key !== ' ' && event.key !== 'Spacebar') return;
    // Space would otherwise scroll the list out from under the header.
    event.preventDefault();
    flSortBy(event.currentTarget.getAttribute('data-sort'));
}

// ---- Buttons ----

function flOnReset() {
    // A mis-tap here throws away a count taken under an alarm, so it is the one
    // action on the page that asks first.
    if (!window.confirm('Clear all evacuation marks?')) return;

    var rows = flRows();
    for (var i = 0; i < rows.length; i++) {
        flSetRow(rows[i], false);
    }
    flUpdateSummary();

    try {
        window.localStorage.removeItem(flStorageKey);
    } catch (e) {
        // Ignored: the screen is already cleared.
    }
}

function flOnReload() {
    // Safe to throw the DOM away: the ticks are in localStorage, and flRestore()
    // puts them straight back on the fresh page. The reload exists to pick up a
    // newer server snapshot, not to clear anything.
    location.reload();
}
