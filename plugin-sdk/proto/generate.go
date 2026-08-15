package proto

//go:generate protoc --proto_path=. --go_out=v1 --go_opt=paths=source_relative --go-grpc_out=v1 --go-grpc_opt=paths=source_relative plugin.proto
