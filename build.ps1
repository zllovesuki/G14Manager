# Fail if we don't have gcc
Get-Command "gcc.exe"
# Fail if we don't have rsrc
Get-Command "rsrc.exe"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOHOSTARCH = "amd64"
$env:CGO_ENABLED = "1"

rsrc.exe -arch amd64 -manifest ROGManager.exe.manifest -ico go.ico -o ROGManager.exe.syso

go build -tags "use_cgo" -ldflags="-H=windowsgui -s -w" .