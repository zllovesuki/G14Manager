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
$LatestTag = (git describe --abbrev=0 --tags)

Write-Host $env:Path

$CommandInstallRSRC = "go get github.com/akavel/rsrc"
Invoke-Expression $CommandInstallRSRC

# easier for us to debug
Get-ChildItem $GOBIN

$RSRC = $GOBIN + "\rsrc.exe -arch amd64 -manifest .\cmd\manager\G14Manager.exe.manifest -ico .\cmd\manager\go.ico -o .\cmd\manager\G14Manager.exe.syso"
Invoke-Expression $RSRC

# Stupid go mod download writes to stderr
$MOD = "go mod download"
Invoke-Expression $MOD 2>&1

Write-Host "Generating static assets"

$PACKED = "go generate .\..."
Invoke-Expression $PACKED

Write-Host "Building prod release"

# $BUILD = "go build -tags 'use_cgo' -ldflags='-H=windowsgui -s -w' ."
$BUILD = "go build -ldflags=`"-H=windowsgui -s -w -X 'main.Version=$LatestTag' -X 'main.IsDebug=no'`" -o build/G14Manager.exe .\cmd\manager"
Invoke-Expression $BUILD

Write-Host "Building debug release"

$BUILD_DEBUG = "go build -gcflags=`"-N -l`" -ldflags=`"-X 'main.Version=$LatestTag-debug' -X 'main.IsDebug=yes'`" -o build/G14Manager.debug.exe .\cmd\manager"
Invoke-Expression $BUILD_DEBUG

Write-Host "Building configurator"

$BUILD_CLIENT = "go build -ldflags=`"-X main.Version=$LatestTag'`" -o build/G14Manager.config.exe .\cmd\client"
Invoke-Expression $BUILD_CLIENT

Write-Host "Building DLLs"

$BUILD_MATRIX_RELEASE_DLL = "MSBuild.exe .\cxx\MatrixController.sln /property:Configuration=Release /property:Platform=x64"
Invoke-Expression $BUILD_MATRIX_RELEASE_DLL