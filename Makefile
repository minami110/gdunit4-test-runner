BINARY := gdunit4-test-runner
CMD := ./cmd/gdunit4-test-runner
VERSION ?= 0.1.0
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-linux build-windows test integration-test lint fmt clean

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 $(CMD)

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe $(CMD)

test:
	go test ./...

integration-test:
	GODOT_PATH=$(GODOT_PATH) go test -v -run TestIntegration ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-windows-amd64.exe
