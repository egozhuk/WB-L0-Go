FROM golang:1.22 AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o myapp ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=build /app/myapp .
COPY web /root/web
COPY migrations /root/migrations
COPY internal/configs /root/configs

EXPOSE 8080

CMD ["./myapp"]