# gather options for tests
TESTARGS=$(TESTOPTIONS)
# gather options for coverage
COVERAGEARGS=$(COVERAGEOPTIONS)
DUMP1090_VERSION=v3.8.1
TAR1090_VERSION=a0491945db41aaa7d49df2951ce1019968048046
READSB_VERSION=e689c9f088321858cb837ae8439975985fecd43a
READSB_REPO=https://github.com/afk11/readsb-protobuf
PROTOBUF_C_VERSION=v1.3.3

readsb-src:
		git clone $(READSB_REPO) readsb-src
		cd readsb-src && git checkout $(READSB_VERSION) && make
install-go-bindata:
		go get -u github.com/jteeuwen/go-bindata/...
install-easyjson:
		go get -u github.com/mailru/easyjson/...
install-protobuf-c:
		git clone https://github.com/protobuf-c/protobuf-c protobuf-c
		cd protobuf-c && git checkout $(PROTOBUF_C_VERSION) && ./autogen.sh && ./configure && make && sudo make install
build-bindata-assets:
		go-bindata -pkg asset -o ./pkg/assets/asset.go assets/...
build-bindata-migrations:
		go-bindata -pkg migrations -o ./pkg/db/migrations/migrations.go -prefix db/migrations db/migrations
build-bindata-migrations-sqlite3:
		go-bindata -pkg migrations_sqlite3 -o ./pkg/db/migrations_sqlite3/migrations.go -prefix db/migrations_sqlite3 db/migrations_sqlite3
build-bindata-migrations-postgres:
		go-bindata -pkg migrations_postgres -o ./pkg/db/migrations_postgres/migrations.go -prefix db/migrations_postgres db/migrations_postgres
build-bindata-dump1090: dump1090
		go-bindata -pkg acmap -o ./pkg/dump1090/acmap/assets.go dump1090/...
build-bindata-tar1090: tar1090
		go-bindata -pkg tar1090 -o ./pkg/tar1090/assets.go tar1090/...
build-bindata-openaip: build-dir-airports
		go-bindata -pkg airports -o ./pkg/airports/assets.go -prefix build/airports build/airports
build-bindata: build-bindata-assets build-bindata-migrations build-bindata-migrations-sqlite3 build-bindata-migrations-postgres build-bindata-dump1090 build-bindata-tar1090 build-bindata-openaip
build-easyjson-adsbx:
		easyjson -all ./pkg/tracker/adsbx_http.go
build-easyjson: build-easyjson-adsbx
build-protobuf:
		protoc -I=./pb/ --go_out=$(GOPATH)/src ./pb/message.proto
delete-build-dir:
		rm -rf build/
build-dir:
		mkdir build/
build-dir-airports: build-dir
		mkdir build/airports/
		go run ./contrib/copy_airport_resources/main.go resources/airports
build-airtrack-linux-amd64: delete-build-dir readsb-src build-bindata build-easyjson build-protobuf
		CGO_ENABLED=1 GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o airtrack.linux-amd64 cmd/airtrack/main.go
build-airtrack-qa-linux-amd64: delete-build-dir readsb-src build-bindata build-easyjson build-protobuf
		CGO_ENABLED=1 GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o airtrackqa.linux-amd64 cmd/airtrack-qa/main.go

test: delete-build-dir readsb-src build-bindata build-easyjson test-cleanup
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
