# Kiyomi — Anti-Bot & Cloudflare Bypass Strategy

This document details Kiyomi's Anti-Bot & Cloudflare bypass architecture, explaining how Kiyomi mimics Mihon's cookie pass-through strategy while extending it with headless solvers and web client pass-through.

---

## 1. How Mihon Solves Cloudflare
Mihon runs on Android devices where the native Android `WebView` (Chromium engine) is available:
1. When an OkHttp request receives a `403 Forbidden` or `503 Service Unavailable` with Cloudflare headers, Mihon triggers an in-app `WebView` window.
2. The user or Chromium's JS engine solves the Cloudflare Turnstile / JavaScript challenge inside the `WebView`.
3. Once passed, Mihon extracts `cf_clearance`, `__cf_bm`, and the `User-Agent` from Android's `CookieManager`.
4. Mihon injects these cookies into its core OkHttp `CookieJar` and attaches the matching `User-Agent` to all subsequent network requests.

---

## 2. Kiyomi's 3-Tier Strategy in Go

Since Kiyomi runs as a Go server, it implements a 3-tier strategy to handle anti-bot protections seamlessly:

### Tier 1: Dynamic Cookie & Header Injection
- `sdk.HttpSource` supports custom `User-Agent`, `Referer`, and domain cookie maps (`isAdult=1`, `readway=2`, `cf_clearance`).
- `h.SetCookies(domainURL, cookieHeader)` allows injecting fresh Cloudflare session cookies into any running provider adapter.
- `sdk.IsCloudflareChallenge(resp, body)` detects Cloudflare `403` / `503` challenge pages (`"Just a moment..."`, Turnstile).

### Tier 2: Headless Solver (CDP / FlareSolverr / Rod)
- For automated headless server setups, Kiyomi can spawn a headless Chrome instance via Chrome DevTools Protocol (`github.com/go-rod/rod` in Go) or delegate to a FlareSolverr sidecar container.
- When a 403/503 is detected, the sidecar loads the page, executes JS, captures `cf_clearance` & `__cf_bm` cookies, and updates Kiyomi's provider cookie jar automatically.
- **TLS Fingerprint Matching**: Spoofs Chrome's TLS Client Hello fingerprint (via Go's `utls`) to prevent Cloudflare from flagging Go's standard `crypto/tls` fingerprint.

### Tier 3: Web Client Interactive Pass-Through
- When a user reads via the Kiyomi Web Client and a provider encounters a Cloudflare challenge:
- The Web Client presents an in-browser prompt allowing the user to solve the challenge directly in their browser.
- The web client posts the resulting cookies and fingerprint back to Kiyomi's `/api/v1/providers/{providerId}/fingerprint` endpoint.
