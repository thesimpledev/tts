test:
	rm -f test/*
	mkdir -p test
	go test ./... -coverprofile=test/coverage.out
	go tool cover -func=test/coverage.out
	go tool cover -html=test/coverage.out -o test/coverage.html
	open test/coverage.html

test-extra:
	rm -f test/*
	mkdir -p test
	go vet ./...
	go test ./... -race -vet=all -shuffle=on -count=1 -timeout=30s -coverprofile=test/coverage.out
	go tool cover -func=test/coverage.out
	go tool cover -html=test/coverage.out -o test/coverage.html
	open test/coverage.html

test-full:
	rm -f test/*
	mkdir -p test
	go fmt ./...
	go vet ./...
	staticcheck ./...
	errcheck ./...
	go test ./... -race -vet=all -shuffle=on -count=1 -timeout=30s -coverprofile=test/coverage.out
	go tool cover -func=test/coverage.out
	go tool cover -html=test/coverage.out -o test/coverage.html
	open test/coverage.html
