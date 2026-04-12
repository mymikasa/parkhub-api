FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/parkhub ./cmd/monolith

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/parkhub /parkhub
COPY configs/ /configs/
EXPOSE 50051 9090 8080
ENTRYPOINT ["/parkhub"]
