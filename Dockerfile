FROM golang:1.26.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN if [ ! -f configs/keys/jwt_private.pem ]; then \
      mkdir -p configs/keys && \
      openssl genpkey -algorithm RSA -out configs/keys/jwt_private.pem -pkeyopt rsa_keygen_bits:2048 && \
      openssl rsa -in configs/keys/jwt_private.pem -pubout -out configs/keys/jwt_public.pem; \
    fi && \
    chmod 644 configs/keys/jwt_private.pem configs/keys/jwt_public.pem
RUN CGO_ENABLED=0 go build -o /bin/parkhub ./cmd/monolith

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/parkhub /parkhub
COPY --from=builder /src/configs/ /configs/
WORKDIR /
EXPOSE 50051 9090 8080
ENTRYPOINT ["/parkhub"]
