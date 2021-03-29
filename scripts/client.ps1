$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOHOSTARCH = "amd64"
$env:CGO_ENABLED = "0"

go run .\cmd\client