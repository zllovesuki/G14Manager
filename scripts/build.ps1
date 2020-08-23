# Fail if we don't have gcc
Get-Command "gcc.exe"
# Fail if we don't have rsrc
Get-Command "rsrc.exe"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CDO_ENABLED = 1

rsrc.exe -arch amd64 -manifest G14Manager.exe.manifest -ico go.ico -o G14Manager.exe.syso

go build -ldflags="-H=windowsgui -s -w" -o "build/G14Manager.exe" .
go build -gcflags="-N -l" -o "build/G14Manager.debug.exe" .