package api

//go:generate protoc --proto_path=. --go_out=./../../internal/api/proto/wms --go_opt=paths=source_relative --go-grpc_out=./../../internal/api/proto/wms --go-grpc_opt=paths=source_relative warehouse.proto
