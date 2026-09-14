// Kiosk behaviour for the tracking page.
//
// Loaded after the inline <script> in reader.html, which declares the globals
// this file reads and the theme poller reassigns:
//   welcomeDurationMs, goodbyeDurationMs, welcomeAnimationUrl,
//   goodbyeAnimationUrl, themeRevision, appVersion, themePollMs.
// Do not redeclare them here.

// ---- Terminal Registration ----
var terminalCredentials = {
    authToken: localStorage.getItem('auth_token'),
    terminalName: localStorage.getItem('terminal_name')
};

function isTerminalRegistered() {
    return terminalCredentials.authToken && terminalCredentials.terminalName;
}

function saveTerminalCredentials(authToken, terminalName) {
    localStorage.setItem('auth_token', authToken);
    localStorage.setItem('terminal_name', terminalName);
    terminalCredentials.authToken = authToken;
    terminalCredentials.terminalName = terminalName;
}

function clearTerminalCredentials() {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('terminal_name');
    terminalCredentials.authToken = null;
    terminalCredentials.terminalName = null;
}

function showRegistrationModal() {
    regModalOpen = true;
    document.getElementById('registrationOverlay').style.display = 'block';
    document.getElementById('regError').style.display = 'none';
    document.getElementById('regConflict').style.display = 'none';
    terminalRegModal.show();
}

function hideRegistrationModal() {
    regModalOpen = false;
    terminalRegModal.hide();
    document.getElementById('registrationOverlay').style.display = 'none';
}

function setRegLoading(loading) {
    var btn = document.getElementById('regSubmitBtn');
    var spinner = document.getElementById('regSpinner');
    var forceBtn = document.getElementById('regForceBtn');
    btn.disabled = loading;
    forceBtn.disabled = loading;
    spinner.style.display = loading ? 'inline' : 'none';
}

function registerTerminal(forceUpdate) {
    var adminLogin = document.getElementById('regAdminLogin').value.trim();
    var adminPassword = document.getElementById('regAdminPassword').value;
    var terminalName = document.getElementById('regTerminalName').value.trim();

    if (!adminLogin || !adminPassword || !terminalName) {
        document.getElementById('regError').textContent = 'All fields are required.';
        document.getElementById('regError').style.display = 'block';
        return;
    }

    setRegLoading(true);
    document.getElementById('regError').style.display = 'none';
    document.getElementById('regConflict').style.display = 'none';

    var xhr = new XMLHttpRequest();
    xhr.open('POST', '/register-terminal', true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onreadystatechange = function() {
        if (xhr.readyState !== 4) return;
        setRegLoading(false);

        if (xhr.status === 200) {
            try {
                var data = JSON.parse(xhr.responseText);
                saveTerminalCredentials(data.auth_token, data.terminal_name);
                hideRegistrationModal();
                rfidInput.focus();
            } catch(e) {
                document.getElementById('regError').textContent = 'Invalid server response.';
                document.getElementById('regError').style.display = 'block';
            }
            return;
        }

        if (xhr.status === 409 && !forceUpdate) {
            document.getElementById('regConflict').style.display = 'block';
            return;
        }

        try {
            var errData = JSON.parse(xhr.responseText);
            document.getElementById('regError').textContent = errData.error || 'Registration failed.';
        } catch(e) {
            document.getElementById('regError').textContent = 'Registration failed.';
        }
        document.getElementById('regError').style.display = 'block';
    };
    xhr.send(JSON.stringify({
        admin_login: adminLogin,
        admin_password: adminPassword,
        terminal_name: terminalName,
        force_update: !!forceUpdate
    }));
}

document.getElementById('terminalRegForm').addEventListener('submit', function(e) {
    e.preventDefault();
    registerTerminal(false);
});

document.getElementById('regForceBtn').addEventListener('click', function() {
    registerTerminal(true);
});

function handleUnauthorized() {
    clearTerminalCredentials();
    showRegistrationModal();
}

// ---- RFID & focus setup (must be before registration check) ----
var rfidInput = document.getElementById('rfidInput');
var cardIdDisplay = document.getElementById('cardIdDisplay');
var readerInstructionImage = document.getElementById('readerInstructionImage');

var regModalOpen = false;
var animationInProgress = false;

// Bootstrap 5 ships its own JS API and no longer depends on jQuery.
// getOrCreateInstance reuses any instance the data-bs-* attributes already
// created for this element instead of building a second one.
var terminalRegModalEl = document.getElementById('terminalRegModal');
var terminalRegModal = bootstrap.Modal.getOrCreateInstance(terminalRegModalEl, {
    backdrop: 'static',
    keyboard: false
});

terminalRegModalEl.addEventListener('shown.bs.modal', function() {
    regModalOpen = true;
});
terminalRegModalEl.addEventListener('hidden.bs.modal', function() {
    regModalOpen = false;
    rfidInput.focus();
    applyPendingThemeState();
});

// ---- On page load: check registration ----
if (!isTerminalRegistered()) {
    showRegistrationModal();
}

var rfidData = '';

rfidInput.addEventListener('keydown', function(event) {
    if (event.key === 'Enter') {
        event.preventDefault();

        rfidData = rfidInput.value.trim();
        if (!rfidData) {
            rfidInput.value = '';
            return;
        }

        if (!isTerminalRegistered()) {
            rfidInput.value = '';
            rfidData = '';
            showRegistrationModal();
            return;
        }

        var payload = {
            visit_key: rfidData,
            signed_in: true
        };

        rfidInput.value = '';
        rfidData = '';

        var xhr = new XMLHttpRequest();
        xhr.open('POST', '/find-and-track', true);
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.setRequestHeader('Terminal-Name', terminalCredentials.terminalName);
        xhr.setRequestHeader('Authorization', 'Bearer ' + terminalCredentials.authToken);
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== 4) return;

            if (xhr.status === 401) {
                handleUnauthorized();
                return;
            }

            if (xhr.status === 404) {
                var alertEl = document.getElementById('error-message');
                alertEl.textContent = 'No visitor data found for the given ID. Please try again or contact the administrator.';
                readerInstructionImage.style.display = 'none';
                alertEl.style.display = 'block';
                setTimeout(function() {
                    alertEl.style.display = 'none';
                    readerInstructionImage.style.display = 'block';
                }, 1500);
                return;
            }

            if (xhr.status !== 200) {
                document.getElementById('student-info').style.display = 'none';
                alert('Error fetching student data');
                return;
            }

            try {
                var data = JSON.parse(xhr.responseText);
            } catch(e) {
                alert('Error parsing server response');
                return;
            }

            readerInstructionImage.style.display = 'none';

            var studentData = data;
            document.getElementById('student-image').src = studentData.visitor.image;
            document.getElementById('student-surname').textContent = studentData.visitor.surname;
            document.getElementById('student-name').textContent = studentData.visitor.name;

            var studentInfo = document.getElementById('student-info');
            var successMessage = document.getElementById('success-message');
            var logoImage = document.getElementById('logo-image');
            var welcomeIcon = document.getElementById('welcome');

            studentInfo.style.display = 'flex';
            successMessage.style.display = 'block';
            logoImage.style.display = 'none';
            welcomeIcon.style.display = 'block';
            animationInProgress = true;
            if (studentData.track_type === 'sign-in') {
                welcomeIcon.src = welcomeAnimationUrl;
            } else {
                welcomeIcon.src = goodbyeAnimationUrl;
            }

            var displayDuration = studentData.track_type === 'sign-in' ? welcomeDurationMs : goodbyeDurationMs;
            setTimeout(function() {
                welcomeIcon.style.display = 'none';
                logoImage.style.display = 'block';
                successMessage.style.display = 'none';
                studentInfo.style.display = 'none';
                readerInstructionImage.style.display = 'block';
                animationInProgress = false;
                applyPendingThemeState();
            }, displayDuration);
        };
        xhr.send(JSON.stringify(payload));
    }
});

window.addEventListener('focus', function() {
    if (!regModalOpen) rfidInput.focus();
});

if (!regModalOpen) rfidInput.focus();

setInterval(function() {
    if (!regModalOpen && document.activeElement !== rfidInput) {
        rfidInput.focus();
    }
}, 1000);

// ---- Theme auto-refresh ----
// This page is a kiosk: it stays open for weeks and never reloads itself, so
// without this poll an admin's theme change would never show up here.

var pendingThemeState = null;
var themeApplyInFlight = false;

function canApplyThemeInPlace() {
    return !animationInProgress;
}

function canReloadPage() {
    return !regModalOpen && !animationInProgress;
}

function preloadImage(url, onDone) {
    if (!url) {
        onDone(false);
        return;
    }
    var img = new Image();
    img.onload = function() { onDone(true); };
    img.onerror = function() { onDone(false); };
    img.src = url;
}

// The favicon is replaced element-and-all: mutating href on the existing
// <link> is ignored by some browsers.
function applyFavicon(url) {
    if (!url) return;
    var existing = document.querySelector('link[rel="icon"]');
    var link = document.createElement('link');
    link.rel = 'icon';
    link.type = 'image/png';
    link.setAttribute('sizes', '16x16');
    link.href = url;
    if (existing && existing.parentNode) {
        existing.parentNode.removeChild(existing);
    }
    document.head.appendChild(link);
}

// applyThemeState swaps in a new theme and reports through onDone whether
// every image it needed actually loaded. Each image is applied as soon as it
// decodes, so a partial failure still moves the terminal as far towards the
// new theme as it can — but the caller is told, so the rest can be retried.
function applyThemeState(state, onDone) {
    // Durations need no download, so they apply unconditionally.
    welcomeDurationMs = state.welcome_duration_ms;
    goodbyeDurationMs = state.goodbye_duration_ms;

    var logoImage = document.getElementById('logo-image');
    // Every image is fetched before it is used: the animations so the next
    // scan does not wait on a download, the logo so the kiosk never shows a
    // blank gap while the file arrives.
    var targets = [
        { url: state.favicon_url, apply: function() {
            applyFavicon(state.favicon_url);
        } },
        { url: state.logo_background_url, apply: function() {
            if (logoImage) logoImage.src = state.logo_background_url;
        } },
        { url: state.welcome_animation_url, apply: function() {
            welcomeAnimationUrl = state.welcome_animation_url;
        } },
        { url: state.goodbye_animation_url, apply: function() {
            goodbyeAnimationUrl = state.goodbye_animation_url;
        } }
    ];

    var loadable = [];
    for (var i = 0; i < targets.length; i++) {
        if (targets[i].url) loadable.push(targets[i]);
    }
    if (loadable.length === 0) {
        onDone(true);
        return;
    }

    var remaining = loadable.length;
    var everythingLoaded = true;
    for (var j = 0; j < loadable.length; j++) {
        (function(target) {
            preloadImage(target.url, function(ok) {
                if (ok) {
                    target.apply();
                } else {
                    everythingLoaded = false;
                }
                remaining -= 1;
                if (remaining === 0) onDone(everythingLoaded);
            });
        })(loadable[j]);
    }
}

function applyPendingThemeState() {
    // themeApplyInFlight stops a retry from racing an apply that is still
    // waiting on its images.
    if (!pendingThemeState || themeApplyInFlight) return;

    if (pendingThemeState.app_version !== appVersion) {
        if (!canReloadPage()) return;
        location.reload();
        return;
    }

    if (!canApplyThemeInPlace()) return;

    var state = pendingThemeState;
    themeApplyInFlight = true;
    applyThemeState(state, function(everythingLoaded) {
        themeApplyInFlight = false;
        if (!everythingLoaded) {
            // An image failed to download. Leave the state pending and the
            // revision untouched so the next poll tries again: recording the
            // revision here would mark the change as handled and strand the
            // terminal on stale artwork until the next theme edit or reload.
            return;
        }
        // A newer state may have arrived while the images were loading; only
        // clear the one that was actually applied.
        if (pendingThemeState === state) {
            pendingThemeState = null;
        }
        themeRevision = state.revision;
    });
}

function handleThemeState(state) {
    if (!state || !state.revision) return;
    if (state.app_version === appVersion && state.revision === themeRevision) {
        return; // nothing changed
    }
    pendingThemeState = state;
    applyPendingThemeState();
}

function pollThemeState() {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/theme-state', true);
    xhr.onreadystatechange = function() {
        if (xhr.readyState !== 4) return;
        // Any failure is ignored on purpose: a server hiccup must never
        // change what the terminal is showing.
        if (xhr.status !== 200) return;
        try {
            handleThemeState(JSON.parse(xhr.responseText));
        } catch (e) {
            // Malformed payload: leave the current theme alone.
        }
    };
    xhr.send();
}

setInterval(pollThemeState, themePollMs);
