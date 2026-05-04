FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY services/iot/go.mod services/iot/go.sum ./services/iot/
RUN go mod download && cd services/iot && go mod download
COPY . .
WORKDIR /src/services/iot
RUN CGO_ENABLED=0 go build -o /bin/parkhub-iot .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/parkhub-iot /parkhub-iot
COPY --from=builder /src/services/iot/config.yaml /config.yaml
WORKDIR /
EXPOSE 50054 9092 8084
ENTRYPOINT ["/parkhub-iot"]
