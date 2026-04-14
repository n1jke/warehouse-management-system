package api

//go:generate protoc --proto_path=. --go_out=./../internal/infrastructure/gen/wms/v1 --go_opt=paths=source_relative --go-grpc_out=./../internal/infrastructure/gen/wms/v1 --go-grpc_opt=paths=source_relative warehouse.proto
