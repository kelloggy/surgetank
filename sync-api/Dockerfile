FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o sync-api .

FROM alpine:3.19
COPY --from=builder /app/sync-api /sync-api
COPY --from=builder /app/quotation.html /quotation.html
EXPOSE 8080
ENTRYPOINT ["/sync-api"]