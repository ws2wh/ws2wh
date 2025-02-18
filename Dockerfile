FROM golang:1.23.6 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ws2wh ./cmd/ws2wh

FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /app/ws2wh .

USER nonroot:nonroot

ENV BACKEND_URL=http://localhost:8080/
ENV REPLY_PATH_PREFIX=/reply
ENV WS_PORT=3000
ENV WS_PATH=/

EXPOSE 3000

CMD ["/app/ws2wh"]
