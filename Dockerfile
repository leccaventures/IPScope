FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ipscope ./cmd/ipscope

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/ipscope /usr/local/bin/ipscope

EXPOSE 9201

ENTRYPOINT ["/usr/local/bin/ipscope", "-config", "/app/config.yml"]
