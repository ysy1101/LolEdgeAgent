# 阶段1：编译
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server cmd/server/main.go

# 阶段2：运行
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
COPY ../configs/ ./configs/
RUN mkdir -p data
EXPOSE 8080
CMD ["./server"]
