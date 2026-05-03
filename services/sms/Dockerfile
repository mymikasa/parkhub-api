FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY services/sms/go.mod services/sms/go.sum ./services/sms/
RUN go mod download && cd services/sms && go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/parkhub-sms ./services/sms

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/parkhub-sms /parkhub-sms
COPY --from=builder /src/services/sms/config.yaml /config.yaml
WORKDIR /
EXPOSE 50053 9093 8082
ENTRYPOINT ["/parkhub-sms"]
