# DNS Override

Kiyomi talks to a lot of remote endpoints: manga metadata sources, chapter
hosts, page image CDNs. Today every one of those connections inherits the
process DNS resolver from the host operating system. On a private network,
behind a captive portal, or under DNS-level censorship, that resolver does
not return useful answers.

This document captures the design for letting operators point Kiyomi at a
custom resolver list — plain DNS, DNS-over-TLS, DNS-over-HTTPS — without
touching code or rebuilding the binary.

---

## Scope

In scope:

- One new env var: `KIYOMI_DNS_RESOLVERS`.
- A new `pkg/dnsresolver` package that speaks plain DNS, DoT, and DoH.
- Plumbing into the two outbound HTTP paths:
  - the **backend image proxy** (`internal/api/handler.go` → `internal/api/proxy_handler.go`),
  - every **plugin subprocess** that calls upstream.
- A default DNS loader in `plugin-sdk/http` so plugin authors get it for free.
- Per-plugin override surface (`WithDNSResolvers`, `WithDNSResolver`).

Out of scope:

- DNSSEC validation (Kiyomi already inherits whatever the resolver validates).
- Cache poisoning defenses (handled by the resolver, not the client).
- Custom resolver selection per source URL (single global list for now).
- Per-OS resolver overrides (`/etc/resolv.conf`, Windows registry, etc.).

---

## Env var

Name: `KIYOMI_DNS_RESOLVERS`.

Value: comma-separated list of resolver URLs. Order preserved; first reachable
upstream wins per lookup.

```
KIYOMI_DNS_RESOLVERS="dns://1.1.1.1,tls://1.1.1.1,https://dns.google/dns-query"
```

URL grammar:

```
<scheme>://<host>[:<port>][/<path>][?bootstrap=<ip>[,<ip>...]]
```

Schemes:

| Scheme | Default port | Path | Notes |
| :--- | :---: | :--- | :--- |
| `dns://` | `53` | n/a | Plain DNS over UDP/TCP. Same wire format Go's `net.Resolver` already speaks. |
| `tls://` | `853` | n/a | DNS-over-TLS (RFC 7858). TCP, TLS-wrapped. |
| `https://` | `443` | `/dns-query` | DNS-over-HTTPS (RFC 8484). HTTP GET with `?dns=` base64url wire payload (POST also accepted). |

Legacy form: bare `host:port` (no scheme) is accepted as `dns://host:port`
for back-compat with the existing `ProviderConfig.DNSResolvers []string`
field at `pkg/provider/sdk/config.go:29-37`.

Bootstrap: when the host of a `tls://` or `https://` URL is a name (not an
IP), it cannot be resolved through itself without a loop. The
`?bootstrap=` query parameter supplies one or more IPs that are resolved
through the system resolver (or the plain-DNS entries in the list) before
the secure connection is opened. If no bootstrap is supplied and the host
is a name, the entry is logged at warn and the system resolver is used as
a one-shot fallback for that single entry.

Empty / unset: behavior is identical to today — the OS-provided resolver
is used everywhere. No crash, no log line.

---

## Resolution flow

```
process start
    │
    ▼
config.Load() reads KIYOMI_DNS_RESOLVERS
    │
    ├─► cfg.DNSResolvers []string
    │
    ▼
api.NewHandler(cfg, lib)
    │
    ├─► proxy httpClient: transport.DialContext = dnsresolver.DialFunc(cfg.DNSResolvers)
    │
    └─► host.NewPluginManager(ManagerOptions{HTTPConfig: cfgToGlobalHTTP(cfg)})
            │
            ▼
        Init(ctx, PluginConfig{HTTPConfig{DNSResolvers, ...}})  [over gRPC]
            │
            ▼
        plugin subprocess
            │
            ▼
        plugin-sdk/http NewClient(WithSDKGlobalHttpConfig(&config.HTTPConfig))
            │
            ▼
        clientConfig.dnsResolvers populated
            │
            ▼
        buildBaseTransport: t.DialContext = dialer with *net.Resolver from pkg/dnsresolver
```

The plugin subprocess also inherits the parent process env via
hashicorp/go-plugin's default subprocess inheritance
(`internal/plugin/host/manager.go:154-161`). `KIYOMI_DNS_RESOLVERS`
therefore reaches the plugin even when the host does not explicitly
forward it via `ManagerOptions.HTTPConfig`. The wire-passed value wins
because explicit beats implicit — the SDK only auto-loads from env when
neither `WithDNSResolvers(...)` nor `WithSDKGlobalHttpConfig(...)`
supplied a value.

---

## Scheme coverage

### Plain DNS (`dns://`)

UDP/TCP port 53. Wire-compatible with Go's `net.Resolver`. Default and
fastest; security depends on the network. Use when on a trusted LAN or
when the resolver itself enforces DNSSEC.

Library: `github.com/miekg/dns`. The `pkg/dnsresolver/plain.go` client
opens a UDP socket (fallback to TCP on truncation) and sends standard
queries.

### DNS-over-TLS (`tls://`)

TCP port 853, TLS-wrapped. Encrypts the query so middleboxes cannot
inspect or rewrite it. Requires either an IP literal in the URL or a
bootstrap IP via `?bootstrap=` to avoid a resolver lookup loop.

Library: `github.com/miekg/dns` over a `tls.Conn`. When a bootstrap is
supplied, TLS verification uses the bootstrap IP but SNI uses the URL
hostname, so a properly-issued certificate for the hostname still
validates.

### DNS-over-HTTPS (`https://`)

TCP port 443, HTTP/2 preferred. RFC 8484. Query sent as base64url wire
format in either `?dns=` GET query or POST body. Default path
`/dns-query` matches Google and Cloudflare.

Library: `net/http` for transport, `github.com/miekg/dns` for wire
encoding. Same bootstrap + SNI rules as DoT.

### When to use what

- **Private LAN**: `dns://` to the internal resolver.
- **Public internet with privacy need**: `tls://` or `https://` to a
  trusted provider.
- **Captive portal / ISP poisoning**: `https://` with bootstrap IP if the
  hostname is being rewritten.

---

## Bootstrap handling

When a DoT or DoH URL points at a hostname, that hostname must be
resolved before the secure connection can be opened. Resolving it
through the same DoT/DoH endpoint creates a loop.

Bootstrap resolution rules:

1. Plain-DNS entries in the resolver list (`dns://...`) try first, in
   declared order. If one of them resolves the hostname, that IP is
   passed to the secure client.
2. If `?bootstrap=<ip>[,<ip>]` is supplied, those IPs are tried next via
   the plain-DNS path.
3. If neither yields an answer, the system resolver is consulted once.
4. If the system resolver also fails, that single entry is logged at
   warn and skipped — the next entry in the list is tried.

The secure client never calls itself, never asks a DoT/DoH entry to
resolve its own hostname. Loop avoidance is the single most important
property of this design.

---

## Plugin SDK surface

`plugin-sdk/http` exposes a default DNS loader plus override options.

### Default behavior

`NewClient(...)` and `NewFingerprintedClient(httpCfg, ...)` automatically
wire a `*net.Resolver` when:

- `WithDNSResolvers(urls)` was passed, OR
- `WithGlobalHttpConfig(*v1.GlobalHttpConfig)` or
  `WithSDKGlobalHttpConfig(sdk.GlobalHttpConfig)` supplied a non-empty
  `DNSResolvers`, OR
- `KIYOMI_DNS_RESOLVERS` is set in the process env.

When none of those are present, the SDK falls through to the system
resolver — identical to today.

### Override

```go
// Plugin author opts in to a specific resolver list:
client := sdkhttp.NewClient(sdkhttp.WithDNSResolvers([]string{
    "tls://1.1.1.1",
    "https://dns.google/dns-query",
}))

// Plugin author supplies a pre-built dialer for full control:
client := sdkhttp.NewClient(sdkhttp.WithDNSResolver(customDialContext))

// Plugin author replaces the entire transport (e.g. for a custom dial chain):
client := sdkhttp.NewClient(sdkhttp.WithTransport(customRoundTripper))
```

Precedence inside the SDK, highest first:

1. `WithTransport(...)` — full replacement, no DNS wiring happens.
2. `WithDNSResolver(fn)` — explicit `DialContext`.
3. `WithDNSResolvers(urls)` — URL list, parsed through `pkg/dnsresolver`.
4. `WithGlobalHttpConfig` / `WithSDKGlobalHttpConfig` — wire-passed list.
5. `os.Getenv("KIYOMI_DNS_RESOLVERS")` — env fallback.
6. `nil` — system resolver.

---

## Backend proxy integration

`internal/api/handler.go:33-47` constructs the image proxy's
`*http.Client`. When `cfg.DNSResolvers` is non-empty, the transport's
`DialContext` is replaced with one built by
`dnsresolver.DialFunc(specs)`. The fingerprint chain (`utls` Client Hello
spoofing) is preserved — only the resolver changes.

The proxy serves page images from upstream CDNs (mangadex uploads,
fanfox/mfcdn, zjcdn). Operators behind a CDN with DNS pinning need this
path to honor their resolver choice without affecting plugin fetches
unless they want both.

---

## Migration from `ProviderConfig.DNSResolvers`

The existing per-provider list at `pkg/provider/sdk/config.go:29-37`
remains functional. Two changes:

- Legacy `host:port` entries are parsed as `dns://host:port` and routed
  through the new `pkg/dnsresolver`.
- When `pkg/provider/sdk.SetGlobalDNSResolvers(urls)` has been called at
  startup (from `cmd/kiyomi/main.go` after `config.Load()`), the global
  list is merged with the per-provider list. Per-provider wins if
  non-empty; global acts as a fallback so an empty provider config still
  honors `KIYOMI_DNS_RESOLVERS`.

No provider-side config file format change. Old TOML with `dns_resolvers
= ["1.1.1.1:53"]` keeps working.

---

## Risks

- **Bootstrap loop.** Mitigation above. Loop avoidance is load-bearing —
  do not weaken it.
- **DoT self-signed certs.** Private-network operators running DoT on
  self-signed certs must supply `?bootstrap=<ip>`; the client then sets
  `InsecureSkipVerify=true` for the secure dial. Cert-pinning support is
  a v2 concern; document the tradeoff in the operator-facing help text.
- **Hostname-only entries without bootstrap.** Single warn log per
  entry, fall through to system resolver for that one entry, do not
  abort startup.
- **Empty / malformed env value.** Warn, do not crash. The system
  resolver is always the safety net.
- **Env vs. wire precedence.** Wire always wins. Operators who need
  per-plugin customization must use the runtime API endpoint
  `POST /api/v1/plugins/:id/config` to push a different
  `GlobalHttpConfig` for that plugin.
- **DoH path templates.** Default `/dns-query`. Operators using
  non-standard paths (e.g. NextDNS) must include the full path in the
  URL: `https://dns.nextdns.io/<profile-id>`.
- **Subnet-local resolvers.** DoT/DoH to a private resolver will fail
  unless the operator's certificate chains to a CA the host trusts. No
  v1 workaround beyond `InsecureSkipVerify` after bootstrap.

---

## See also

- [`dns_override_sequence.md`](./dns_override_sequence.md) — sequence diagram.
- [`provider_plugin_architecture.md`](./provider_plugin_architecture.md) — how plugins reach upstream.
- [`../plugin_developer/dns_overrides.md`](../plugin_developer/dns_overrides.md) — plugin author guide.
