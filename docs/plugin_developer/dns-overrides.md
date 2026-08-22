# DNS Overrides for Plugin Authors

The Kiyomi plugin SDK ships a default DNS loader. Every plugin that uses
`sdkhttp.NewClient` gets DNS override plumbing for free. You only need
this guide if you want to opt out, opt in differently, or troubleshoot
DNS in local development.

---

## Default behavior

If you write:

```go
client := sdkhttp.NewClient(
    sdkhttp.WithSDKGlobalHttpConfig(&config.HTTPConfig),
    sdkhttp.WithTimeout(20 * time.Second),
).StandardClient()
```

…the SDK automatically:

1. Reads `DNSResolvers` from the wire-passed `config.HTTPConfig`.
2. If that list is empty, reads `KIYOMI_DNS_RESOLVERS` from the process
   environment.
3. If both are empty, uses the system resolver (current behavior — no
   change for you).

No per-plugin code is needed. The host operator flips DNS via env var or
the runtime API endpoint, and your plugin inherits the new resolver.

---

## Opt-in override

If your plugin needs a specific resolver independent of the host's
config — for example, to talk to a private upstream that requires DoH —
pass `WithDNSResolvers` last. It wins over the wire-passed config:

```go
client := sdkhttp.NewClient(
    sdkhttp.WithSDKGlobalHttpConfig(&config.HTTPConfig),
    sdkhttp.WithDNSResolvers([]string{
        "https://dns.internal.example/dns-query",
    }),
    sdkhttp.WithTimeout(20 * time.Second),
).StandardClient()
```

The list is parsed by `pkg/dnsresolver.ParseList`. Accepted forms:

- `dns://1.1.1.1`
- `dns://1.1.1.1:5353`
- `tls://1.1.1.1`
- `tls://one.one.one.one?bootstrap=1.1.1.1,1.0.0.1`
- `https://dns.google/dns-query`
- `https://dns.nextdns.io/abc123`
- `1.1.1.1:53` (legacy form, treated as `dns://1.1.1.1:53`)

---

## Full transport replacement

If you need fine-grained control — for example, to inject a custom
dialer that combines DNS override with a connection-pool tuning knob —
replace the transport entirely:

```go
customTransport := &http.Transport{
    Proxy: http.ProxyFromEnvironment,
    DialContext: myCustomDialContext,
    // ... other knobs
}

client := sdkhttp.NewClient(
    sdkhttp.WithTransport(customTransport),
).StandardClient()
```

`WithTransport` bypasses the SDK's DNS wiring completely. You're
responsible for the dialer.

---

## Finer dialer control

If you want the SDK to handle transport layering (retry, rate limit,
headers, TLS fingerprinting) but you want to supply the dialer
yourself, use `WithDNSResolver`:

```go
client := sdkhttp.NewClient(
    sdkhttp.WithDNSResolver(myDialContext),
    sdkhttp.WithTLSProfile(sdkhttp.TLSProfileChrome),
).StandardClient()
```

`WithDNSResolver` wins over `WithDNSResolvers`, which wins over the
wire-passed list, which wins over the env var.

---

## Reading config from the host

The wire-passed list arrives in `Init`:

```go
func (p *MyPlugin) Init(ctx context.Context, config sdk.PluginConfig) error {
    resolvers := config.HTTPConfig.DNSResolvers // []string, may be empty
    if len(resolvers) > 0 {
        slog.Info("using host-supplied dns resolvers",
            "plugin", p.id,
            "count", len(resolvers),
        )
    }
    // ...
}
```

Treat it as a hint, not a directive. You may pass `WithDNSResolvers` to
override it for your own fetches.

---

## Testing locally

Run your plugin binary standalone with the env var set:

```bash
KIYOMI_DNS_RESOLVERS=tls://1.1.1.1 \
    go run ./cmd/my-plugin
```

The plugin subprocess inherits the env from your shell, so the SDK's
auto-load path picks it up the same way it does in production.

For an isolated test, point at a local DoH stub:

```bash
KIYOMI_DNS_RESOLVERS=https://127.0.0.1:8443/dns-query \
    go test ./...
```

Pair with `httptest.NewServer` running a DoH handler in your test
fixtures — see `pkg/dnsresolver/dnsresolver_test.go` for the pattern.

---

## Examples

### 1. Default — no plugin changes needed

```go
// plugin.go
client := sdkhttp.NewClient(
    sdkhttp.WithSDKGlobalHttpConfig(&config.HTTPConfig),
).StandardClient()
```

### 2. Override with a private resolver

```go
// plugin.go — for a plugin that talks to an internal catalog
client := sdkhttp.NewClient(
    sdkhttp.WithSDKGlobalHttpConfig(&config.HTTPConfig),
    sdkhttp.WithDNSResolvers([]string{
        "https://dns.internal.corp/dns-query",
    }),
).StandardClient()
```

### 3. Full replacement for advanced routing

```go
// plugin.go — for a plugin that needs its own dialer
transport := &http.Transport{
    Proxy: nil,
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        return customDialer.DialContext(ctx, network, addr)
    },
    // ... pool, timeouts, etc.
}

client := sdkhttp.NewClient(
    sdkhttp.WithTransport(transport),
).StandardClient()
```

---

## See also

- [`../design/dns-override.md`](../design/dns-override.md) — full design spec.
- [`../design/dns-override-sequence.md`](../design/dns-override-sequence.md) — sequence diagram.
- [`../design/provider_plugin_architecture.md`](../design/provider_plugin_architecture.md) — plugin lifecycle and config flow.
