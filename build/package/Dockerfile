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

############################
FROM scratch

LABEL app_name="autowp.goautowp" \
      maintainer="dmitry@pereslegin.ru"

EXPOSE 8080

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/ImageMagick-7 /etc/ImageMagick-7
COPY --from=builder /lib/ld-musl-x86_64.so.1 /lib/libz.so.1 /lib/
COPY --from=builder \
    /usr/lib/libMagickWand-7.Q16HDRI.so.9 \
    /usr/lib/libMagickCore-7.Q16HDRI.so.9 \
    /usr/lib/libX11.so.6 \
    /usr/lib/liblcms2.so.2 \
    /usr/lib/libxml2.so.2 \
    /usr/lib/libfontconfig.so.1 \
    /usr/lib/libfreetype.so.6 \
    /usr/lib/libXext.so.6 \
    /usr/lib/libbz2.so.1 \
    /usr/lib/libltdl.so.7 \
    /usr/lib/libxcb.so.1 \
    /usr/lib/liblzma.so.5 \
    /usr/lib/libexpat.so.1 \
    /lib/libuuid.so.1 \
    /usr/lib/libpng16.so.16 \
    /usr/lib/libbrotlidec.so.1 \
    /usr/lib/libXau.so.6 \
    /usr/lib/libXdmcp.so.6 \
    /usr/lib/libbrotlicommon.so.1 \
    /usr/lib/libbsd.so.0 \
    /usr/lib/libmd.so.0 \
    /usr/lib/

COPY --from=builder /app /app
COPY migrations /migrations
COPY traffic-migrations /traffic-migrations
COPY auth/migrations /auth/migrations
COPY defaults.yaml /defaults.yaml

USER appuser:appuser

ENTRYPOINT ["/app"]
