# time-table-back-dev2

Lambda function
1. create function -> author from scratch -> set your Function name and runtime -> use an existing role ->  LabRole
2) edit properties of function
   Code -> set Handler as "main"
   Config -> Triggers -> add your API gateway as trigger

	 
API Gateway 
1. build HTTP API
2. Add route as POST /timetable
3. edit CORS
   Access-Control-Allow-Origin = *
   Access-Control-Allow-Headers = true
   Access-Control-Allow-Methods = POST
   Access-Control-Expose-Headers = filename

S3 Bucket 
1. properties -> static website hosting = Enable
2. Permission -> set Block all public access to False
3. set policy as
   {
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "PublicReadGetObject",
            "Effect": "Allow",
            "Principal": "*",
            "Action": "s3:GetObject",
   			   "Resource": "arn:aws:s3:::{bucketName}/*"
        }
    ]
}
upload index.html and SciFirstYearClass - 66_1.csv as sample
build file in powershell : 
$env:GOOS = "linux"
$env:CGO_ENABLED = "0"
$env:GOARCH = "amd64"
go build -o main back.go
then upload zip file into function

http://timetablebuckettt.s3-website-us-east-1.amazonaws.com/
