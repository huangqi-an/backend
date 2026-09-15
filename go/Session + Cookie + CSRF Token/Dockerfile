# syntax=docker/dockerfile:1

# ---- build: 编译 Go 二进制 ----
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server .

# ---- runtime: 精简运行镜像 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /server .
EXPOSE 3000
CMD ["./server"]