aoclb:
	@export CGO_ENABLED=1
	go build -v -o ./.dist/ ./cmd/aoclb/
	@.dist/aoclb
