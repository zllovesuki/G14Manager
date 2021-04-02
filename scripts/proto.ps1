Write-Host "Installing protoc-gen for Go"
go get -u github.com/golang/protobuf/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc

Write-Host "Installing protoc-gen for TypeScript"
Set-Location .\client
npm install ts-protoc-gen
Set-Location ..
Get-ChildItem .\client\node_modules\.bin

Write-Host "Generating proto definitions for Go"
protoc.exe --go_opt=paths=source_relative --go_out=. --go-grpc_out=. --go-grpc_opt=paths=source_relative rpc/protocol/*.proto

Write-Host "Generating proto definitions for TypeScript"
protoc.exe --plugin=protoc-gen-ts=.\client\node_modules\.bin\protoc-gen-ts.cmd --proto_path=. --js_out=import_style="commonjs,binary:./client/src" --ts_out=service=grpc-web:client/src rpc/protocol/*.proto