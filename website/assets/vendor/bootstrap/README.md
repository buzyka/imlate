# Bootstrap (vendored)

**Version:** 5.3.8
**Upstream:** https://getbootstrap.com/
**License:** MIT — https://github.com/twbs/bootstrap/blob/main/LICENSE

## Why these files are committed

The tracking page (`website/reader.html`) runs full-screen in a browser kiosk on
Raspberry Pi panels. Those panels sit on a LAN that may have no route to the
internet, and the page stays open for weeks. Loading Bootstrap from a CDN made a
CDN outage — or a network without internet access — able to leave the terminal
unstyled and its registration modal dead. Serving the files from this binary's
own `/assets` tree removes that failure mode entirely.

`bootstrap.bundle.min.js` includes Popper, so it replaces the three separate
scripts (jQuery, Popper, Bootstrap) the page used to load. Bootstrap 5 has no
jQuery dependency at all.

## Files

| File | Purpose |
| --- | --- |
| `bootstrap.min.css` | Compiled CSS |
| `bootstrap.bundle.min.js` | Compiled JS including Popper |

Source maps (`*.map`) are deliberately not vendored: they add ~700 KB to the
image and are only useful in DevTools.

## How to update

```sh
BS_VERSION=5.3.8
curl -fsSL "https://cdn.jsdelivr.net/npm/bootstrap@${BS_VERSION}/dist/css/bootstrap.min.css" \
  -o website/assets/vendor/bootstrap/bootstrap.min.css
curl -fsSL "https://cdn.jsdelivr.net/npm/bootstrap@${BS_VERSION}/dist/js/bootstrap.bundle.min.js" \
  -o website/assets/vendor/bootstrap/bootstrap.bundle.min.js
```

Then bump the version at the top of this file and run `go test ./cmd/app/...`.
