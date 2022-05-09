SHORT_NAME ?= exporter_care

BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
HASH = $(shell git describe --dirty --tags --always)
VERSION ?= unknown
REPO = github.com/fengxsong/exporter-care

BUILD_PATH = main.go
OUTPUT_PATH = build/_output/bin/$(SHORT_NAME)

LDFLAGS := -s -X ${REPO}/cmd.buildDate=${BUILD_DATE} \
	-X ${REPO}/cmd.gitCommit=${HASH} \
	-X ${REPO}/cmd.version=${VERSION}

tidy:
	go mod tidy

vendor: tidy
	go mod vendor

bin:
	CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o ${OUTPUT_PATH} ${BUILD_PATH} || exit 1

linux-bin:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o ${OUTPUT_PATH} ${BUILD_PATH} || exit 1

upx:
	upx ${OUTPUT_PATH}
