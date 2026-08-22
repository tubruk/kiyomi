module github.com/tubruk/kiyomi

go 1.26.4

require (
	github.com/PuerkitoBio/goquery v1.12.0
	github.com/hashicorp/go-plugin v1.8.0
	github.com/joho/godotenv v1.5.1
	github.com/labstack/echo/v4 v4.15.4
	github.com/miekg/dns v1.1.73
	github.com/refraction-networking/utls v1.8.2
	github.com/stretchr/testify v1.12.1
	github.com/tubruk/kiyomi/plugin-sdk v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.83.1
)

require (
	github.com/fatih/color v1.19.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/oklog/run v1.2.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260819154853-08b0e4226688 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

require (
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/andybalholm/cascadia v1.3.4 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/labstack/gommon v0.5.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0
	golang.org/x/sync v0.22.0
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/tubruk/kiyomi/plugin-sdk => ./plugin-sdk
