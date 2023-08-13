# time-table-back-dev2
build file in powershell : $env:GOOS = "linux"
$env:CGO_ENABLED = "0"
$env:GOARCH = "amd64"
go build -o main back.go

http://timetablebuckettt.s3-website-us-east-1.amazonaws.com/
