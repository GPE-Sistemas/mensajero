.PHONY: test generar vista-previa

test:
	go vet ./...
	go test -race ./...

generar:
	go generate ./...

# make vista-previa P=gas/reset-password
vista-previa:
	open "$$(go run ./cmd/mensajero vista-previa $(P))"
