build:
	go build -race -o bin/wgk8s cmd/wgk8s/wgk8s.go
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/wgcni cmd/wgcni/wgcni.go

test:
	go test -cover -v ./...
