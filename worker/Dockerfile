FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o worker .

FROM alpine:3.19
COPY --from=builder /app/worker /worker
COPY --from=builder /app/quotation.html /quotation.html
ENTRYPOINT ["/worker"]