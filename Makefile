# gather options for tests
TESTARGS=$(TESTOPTIONS)
# gather options for coverage
COVERAGEARGS=$(COVERAGEOPTIONS)
DUMP1090_VERSION=v3.8.1
TAR1090_VERSION=a0491945db41aaa7d49df2951ce1019968048046

install-go-bindata:
		go get -u github.com/jteeuwen/go-bindata/...
build-bindata-assets:
		go-bindata -pkg asset -o ./pkg/assets/asset.go assets/...
build-bindata-migrations:
		go-bindata -pkg migrations -o ./pkg/db/migrations/migrations.go -prefix db/migrations db/migrations
build-bindata-dump1090: dump1090
		go-bindata -pkg acmap -o ./pkg/dump1090/acmap/assets.go dump1090/...
build-bindata-tar1090: tar1090
		go-bindata -pkg tar1090 -o ./pkg/tar1090/assets.go tar1090/...
build-protobuf:
		protoc -I=./pb/ --go_out=$(GOPATH)/src ./pb/message.proto
build-airtrack-linux-amd64: build-bindata-assets build-bindata-migrations build-bindata-dump1090 build-bindata-tar1090 build-protobuf
		GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o airtrack.linux-amd64 cmd/airtrack/main.go
build-airtrack-qa-linux-amd64: build-bindata-assets build-bindata-migrations build-bindata-dump1090 build-bindata-tar1090 build-protobuf
		GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o airtrackqa.linux-amd64 cmd/airtrack-qa/main.go

test: test-cleanup
	go test -coverprofile=./coverage/tests.out ./... \
	$(TESTARGS)

# test cleanup, remove old coverage
test-cleanup:
	rm -rf coverage/ 2>> /dev/null || exit 0 && \
	mkdir coverage/

# concat all coverage reports together
coverage-concat:
	echo "mode: set" > coverage/full && \
	grep -h -v "^mode:" coverage/*.out >> coverage/full

# full coverage report
coverage: coverage-concat
	go tool cover -func=coverage/full $(COVERAGEARGS)

# full coverage report
coverage-html: coverage-concat
	go tool cover -html=coverage/full $(COVERAGEARGS)

dump1090:
	./contrib/refresh_dump1090.sh "${DUMP1090_VERSION}"

tar1090:
	./contrib/refresh_tar1090.sh "${TAR1090_VERSION}"
