FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o async-api .

FROM alpine:3.19
COPY --from=builder /app/async-api /async-api
EXPOSE 8080
ENTRYPOINT ["/async-api"]