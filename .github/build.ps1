# https://github.com/actions/virtual-environments/blob/main/images/win/Windows2019-Readme.md

# should have Mingw-w64 installed already
Get-Command "gcc.exe"

$CommandInstallRSRC = "go get github.com/akavel/rsrc"
Invoke-Expression $CommandInstallRSRC

# Fail if we don't have rsrc
Get-Command "rsrc.exe"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOHOSTARCH = "amd64"
$env:CGO_ENABLED = "1"

go build -tags "use_cgo" -ldflags="-H=windowsgui -s -w" .