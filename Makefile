proto:
	docker build -t protogen -f Dockerfile.protogen .
	rm -rf rpc/protocol/*.pb.go
	docker run -v `pwd`:/proto protogen --go_opt=paths=source_relative --go_out=. \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		rpc/protocol/*.proto