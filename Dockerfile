FROM golang:1.23 as builder

WORKDIR /app

COPY go.mod  .

COPY go.sum .

RUN go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build -o findme .


FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/findme .

EXPOSE 8080

CMD [ "./findme" ]
