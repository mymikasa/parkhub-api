FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY services/identity/go.mod services/identity/go.sum ./services/identity/
RUN go mod download && cd services/identity && go mod download
COPY . .
WORKDIR /src/services/identity
RUN CGO_ENABLED=0 go build -o /bin/parkhub-identity .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/parkhub-identity /parkhub-identity
COPY --from=builder /src/services/identity/config.yaml /config.yaml
WORKDIR /
EXPOSE 50052 9092 8081
ENTRYPOINT ["/parkhub-identity"]
