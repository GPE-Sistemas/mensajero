FROM golang:1.26-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /mensajero ./cmd/mensajero

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /mensajero /mensajero
EXPOSE 8080
ENTRYPOINT ["/mensajero"]
