FROM golang:1.23-alpine

WORKDIR /app

COPY main.go .
COPY go.mod .

RUN go build -o my-go-app .

CMD ["./my-go-app"]