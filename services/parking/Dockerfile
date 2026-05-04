FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY services/parking/go.mod services/parking/go.sum ./services/parking/
RUN go mod download && cd services/parking && go mod download
COPY . .
WORKDIR /src/services/parking
RUN CGO_ENABLED=0 go build -o /bin/parkhub-parking .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/parkhub-parking /parkhub-parking
COPY --from=builder /src/services/parking/config.yaml /config.yaml
WORKDIR /
EXPOSE 50055 9092 8085
ENTRYPOINT ["/parkhub-parking"]
