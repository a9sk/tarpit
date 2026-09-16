.PHONY: all build certs run test clean

all: build

build:
	go build .

certs:
	mkdir -p certs
	openssl req -x509 -newkey rsa:4096 \
		-keyout certs/localhost.key \
		-out certs/localhost.crt \
		-days 365 \
		-nodes \
		-subj "/CN=localhost"

run: build
	./tarpit

test:
	go test ./...

clean:
	rm -f tarpit
