# prediction-go

### Generate Stubs
```
protoc --go_out=. --go-grpc_out=. prediction.proto
```
### Download packages
```
go mod init prediction
go mod tidy
```