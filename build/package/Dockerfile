FROM golang:1.15-alpine as builder

RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR $GOPATH/src/github.com/autowp/goautowp/

# Populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .
RUN go mod download && \
    go mod verify

COPY . $GOPATH/src/github.com/autowp/goautowp/

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /app cmd/goautowp/goautowp.go

############################
FROM scratch

LABEL app_name="autowp.goautowp" \
      maintainer="dmitry@pereslegin.ru"

EXPOSE 8080

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /app /app
COPY migrations /migrations
COPY defaults.yaml /defaults.yaml

USER appuser:appuser

ENTRYPOINT ["/app"]
