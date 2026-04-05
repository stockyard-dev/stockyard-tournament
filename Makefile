build:
	CGO_ENABLED=0 go build -o tournament ./cmd/tournament/

run: build
	./tournament

test:
	go test ./...

clean:
	rm -f tournament

.PHONY: build run test clean
