
install-go-bindata:
		go get -u github.com/jteeuwen/go-bindata/...
build-bindata-assets:
		go-bindata -pkg asset -debug -o ./pkg/assets/asset.go assets/...
build-bindata-migrations:
		go-bindata -pkg migrations -debug -o ./pkg/db/migrations/migrations.go -prefix db/migrations db/migrations
build-protobuf:
		protoc -I=./pb/ --go_out=$(GOPATH)/src ./pb/message.proto
build-airtrack: build-bindata-assets build-bindata-migrations build-protobuf
		GO111MODULE=on go build -o airtrack cmd/airtrack/main.go
build-airtrack-qa: build-bindata-assets build-bindata-migrations build-protobuf
		GO111MODULE=on go build -o airtrackqa cmd/airtrack-qa/main.go
