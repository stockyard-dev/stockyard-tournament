FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go mod download && CGO_ENABLED=0 go build -o tournament ./cmd/tournament/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/tournament .
ENV PORT=9804 DATA_DIR=/data
EXPOSE 9804
CMD ["./tournament"]
