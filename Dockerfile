FROM golang:1.23-alpine as builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod  .

COPY go.sum .

RUN go mod download

COPY . .

RUN GOMAXPROCS=1 GOMEMLIMIT=450MiB go build -o findme .


FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/findme .

EXPOSE 8080

CMD [ "./findme" ]
