.PHONY: proto openapi lint test

PROTOC_ARGS=--proto_path=proto --proto_path=proto/third_party/googleapis --proto_path=proto/third_party/grpc-gateway
proto: openapi
	@echo "Generating gRPC and Gateway code..."
	@protoc $(PROTOC_ARGS) \
	 --go_out=./proto --go_opt=paths=source_relative \
	 --go-grpc_out=./proto --go-grpc_opt=paths=source_relative \
	 --grpc-gateway_out=./proto --grpc-gateway_opt=paths=source_relative \
	 proto/*.proto
	@echo "Done."

openapi:
	@echo "Generating OpenAPIv2 specification..."
	@protoc $(PROTOC_ARGS) \
	 --openapiv2_out ./proto \
	 --openapiv2_opt logtostderr=true \
	 proto/*.proto
	@echo "Done."

lint: 
	@echo "Running linter..."
	@golangci-lint run ./api-gateway/... ./assets/... ./pkg/... ./proto/... ./users/... ./wallets/...
	@echo "Linter finished."

test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Tests finished"
