# Fail if we don't have gcc
Get-Command "gcc.exe"
# Fail if we don't have rsrc
Get-Command "rsrc.exe"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = 1

rsrc.exe -arch amd64 -manifest .\cmd\manager\G14Manager.exe.manifest -ico .\cmd\manager\go.ico -o .\cmd\manager\G14Manager.exe.syso

go get golang.org/x/tools/cmd/stringer
go generate .\...
go build -ldflags="-H=windowsgui -s -w -X 'main.Version=v0.0.0-staging' -X 'main.IsDebug=no'" -o "build/G14Manager.exe" .\cmd\manager
go build -gcflags="-N -l" -ldflags="-X 'main.Version=v0.0.0-debug' -X 'main.IsDebug=yes'" -o "build/G14Manager.debug.exe" .\cmd\manager

go build -gcflags="-N -l" -o "build/G14Manager.config.exe" .\cmd\client