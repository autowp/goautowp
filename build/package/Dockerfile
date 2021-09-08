FROM golang:1.17-alpine as builder

RUN apk update && apk add --no-cache gcc git ca-certificates imagemagick imagemagick-dev libc-dev xmlstarlet && update-ca-certificates

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
COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify

COPY . $GOPATH/src/github.com/autowp/goautowp/

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app cmd/goautowp/goautowp.go

RUN cat /etc/ImageMagick-7/policy.xml | \
        xmlstarlet ed -u "/policymap/policy[@domain='resource'][@name='memory']/@value" -v "2GiB" | \
        xmlstarlet ed -u "/policymap/policy[@domain='resource'][@name='disk']/@value" -v "10GiB" > /etc/ImageMagick-7/policy2.xml && \
    cat /etc/ImageMagick-7/policy2.xml > /etc/ImageMagick-7/policy.xml

RUN ls -lah /etc

############################
FROM scratch

LABEL app_name="autowp.goautowp" \
      maintainer="dmitry@pereslegin.ru"

EXPOSE 8080

USER appuser:appuser

ENTRYPOINT ["/app"]

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/fonts /etc/fonts
COPY --from=builder /etc/ssl /etc/ssl
COPY --from=builder /etc/ImageMagick-7 /etc/ImageMagick-7
COPY --from=builder /usr/lib /usr/lib
COPY --from=builder /usr/share /usr/share
COPY --from=builder /lib /lib
COPY --from=builder /app /app

COPY auth/migrations /auth/migrations
COPY migrations /migrations
COPY traffic-migrations /traffic-migrations
COPY defaults.yaml /defaults.yaml
