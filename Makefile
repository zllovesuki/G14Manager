proto_windows:
	go get -u github.com/golang/protobuf/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc
	rm -rf rpc/protocol/*.pb.go
	protoc.exe --go_opt=paths=source_relative --go_out=. \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		rpc/protocol/*.proto

proto_wsl:
	docker build -t protogen -f Dockerfile.wsl.protogen .
	rm -rf rpc/protocol/*.pb.go
	docker run -v `pwd`:/proto protogen --go_opt=paths=source_relative --go_out=. \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		rpc/protocol/*.proto