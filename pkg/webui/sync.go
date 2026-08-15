package webui

// sync.go holds the go:generate directive that copies the freshly built
// web bundle from web/dist into pkg/webui/dist/ so //go:embed can pick
// it up. Run with `go generate ./pkg/webui/...` from the repo root after
// `cd web && bun run build`.

//go:generate bash ../../scripts/sync-webui.sh
