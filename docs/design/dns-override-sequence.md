# DNS Override — Resolution Sequence

End-to-end sequence from `KIYOMI_DNS_RESOLVERS` env var through the
plugin subprocess's first outbound HTTP request. Mirrors the prose in
[`dns-override.md`](./dns-override.md).

---

## Startup — main app

```
  main()                                  config.Load()
    │                                           │
    │                                           ▼
    │                              parse KIYOMI_DNS_RESOLVERS
    │                              → cfg.DNSResolvers []string
    │                                           │
    ▼                                           │
  api.NewHandler(cfg, lib)                      │
    │                                           │
    ├─► proxy client                             │
    │     transport.DialContext = dnsresolver    │
    │       .DialFunc(cfg.DNSResolvers)          │
    │                                           │
    └─► host.NewPluginManager({                  │
    │     HTTPConfig: cfgToGlobalHTTP(cfg), ────┘
    │     PluginDir: cfg.PluginDir,
    │     Registry:  reg,
    │   })
```

## Plugin subprocess launch

```
  manager.go:LaunchPlugin
    │
    ▼
  plugin.NewClient(&plugin.ClientConfig{
      Cmd: exec.Command(absPath),    // inherits parent env
      HandshakeConfig: sdk.HandshakeConfig,
      Plugins:         sdk.PluginMap(...),
  })
    │
    ▼
  plugin subprocess starts → reads env (including KIYOMI_DNS_RESOLVERS)
```

## Init via gRPC

```
  host                              plugin subprocess
   │                                       │
   │  Init(ctx, PluginConfig{              │
   │    HTTPConfig: GlobalHttpConfig{      │
   │      DNSResolvers: cfg.DNSResolvers, ─┼─► received as
   │      ...                               │   config.HTTPConfig.DNSResolvers
   │    },                                 │
   │  })                                   │
   │                                       ▼
   │                              plugin.Init(ctx, config)
   │                                       │
   │                                       ▼
   │                              plugin constructs HTTP client:
   │                                sdkhttp.NewClient(
   │                                  sdkhttp.WithSDKGlobalHttpConfig(
   │                                    &config.HTTPConfig,
   │                                  ),
   │                                )
   │                                       │
   │                                       ▼
   │                              clientConfig.dnsResolvers = config.HTTPConfig.DNSResolvers
   │                                       │
   │                                       ▼
   │                              buildBaseTransport:
   │                                if len(cfg.dnsResolvers) > 0:
   │                                  dialer.Resolver = &net.Resolver{
   │                                    PreferGo: true,
   │                                    Dial: dnsresolver.DialFuncFromURLs(specs),
   │                                  }
   │                                  t.DialContext = dialer.DialContext
   │                                       │
   │                                       ▼
   │                              client ready
   │                              ←─── nil ──── (Init returns)
```

## First upstream request from plugin

```
  plugin.doRequest("https://api.mangadex.org/manga/...")
    │
    ▼
  client.Do(req)
    │
    ▼
  transport.DialContext(ctx, "tcp", "api.mangadex.org:443")
    │
    ▼
  dialer.Resolver.LookupHost(ctx, "api.mangadex.org")
    │
    ▼
  pkg/dnsresolver.DialFuncFromURLs(specs) → ctx, "udp", "1.1.1.1:53"
    │
    ▼
  ┌─ scheme = "dns://" → plain DNS via miekg
  ├─ scheme = "tls://" → DoT via miekg over tls.Conn
  └─ scheme = "https://" → DoH via net/http with base64url wire
    │
    ▼
  response: api.mangadex.org → A 104.21.0.1
    │
    ▼
  dialer dials 104.21.0.1:443 with TLS fingerprint
    │
    ▼
  HTTP request lands
```

## Env fallback path (host did not populate HTTPConfig)

```
  If config.HTTPConfig.DNSResolvers is empty:
    │
    ▼
  NewClient default loader:
    if cfg.dnsResolvers == nil && cfg.customDialContext == nil:
      cfg.dnsResolvers = dnsresolver.LoadFromEnv("KIYOMI_DNS_RESOLVERS")
    │
    ▼
  buildBaseTransport proceeds identically
```

Env inheritance from main app to plugin subprocess is automatic via
hashicorp/go-plugin's default subprocess env passthrough
(`internal/plugin/host/manager.go:154-161`). No explicit `Env` field is
required.

---

## Precedence summary

In a plugin subprocess, the resolver list is resolved in this order:

1. `sdkhttp.WithDNSResolver(fn)` Option — custom `DialContext`.
2. `sdkhttp.WithDNSResolvers(urls)` Option — explicit URL list.
3. `WithGlobalHttpConfig` / `WithSDKGlobalHttpConfig` — wire-passed
   `config.HTTPConfig.DNSResolvers` from the host.
4. `os.Getenv("KIYOMI_DNS_RESOLVERS")` — env fallback.
5. System resolver (Go default).

Wire beats env. Explicit Option beats wire. Custom dialer beats Option.

---

## See also

- [`dns-override.md`](./dns-override.md) — full design spec.
- [`../plugin_developer/dns-overrides.md`](../plugin_developer/dns-overrides.md) — plugin author guide.
