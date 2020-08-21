# https://github.com/actions/virtual-environments/blob/main/images/win/Windows2019-Readme.md

# should have Mingw-w64 installed already
Get-Command "gcc.exe"

$env:GOPATH = $env:GITHUB_WORKSPACE + "\go"
$GOBIN = $env:GOPATH + "\bin"
$env:Path += ";" + $env:GOBIN
$env:GOOS = "windows"
$env:GOARCH = "amd64"
# $env:GOHOSTARCH = "amd64"
$env:CGO_ENABLED = "1"

Write-Host $env:Path

$CommandInstallRSRC = "go get github.com/akavel/rsrc"
Invoke-Expression $CommandInstallRSRC

# easier for us to debug
Get-ChildItem $GOBIN

$RSRC = $GOBIN + "\rsrc.exe -arch amd64 -manifest ROGManager.exe.manifest -ico go.ico -o ROGManager.exe.syso"
Invoke-Expression $RSRC

# Stupid go mod download writes to stderr
$MOD = "go mod download"
Invoke-Expression $MOD 2>&1

Write-Host "Building prod release"

# $BUILD = "go build -tags 'use_cgo' -ldflags='-H=windowsgui -s -w' ."
$BUILD = "go build -ldflags=`"-H=windowsgui -s -w -X 'main.Version=$env:GITHUB_REF'`" -o build/ROGManager.exe ."
Invoke-Expression $BUILD

Write-Host "Building debug release"

$BUILD_DEBUG = "go build -ldflags=`"-X 'main.Version=$env:GITHUB_REF'`" -o build/ROGManager.debug.exe ."
Invoke-Expression $BUILD_DEBUG