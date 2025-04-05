FROM docker.io/library/alpine:edge AS builder

ENV HOME=/root
ENV PATH=$PATH:/usr/local/go/bin:$HOME/go/bin \
    GOPATH=$HOME/go

RUN apk --update --no-cache add \
        aom-libs \
        ca-certificates \
        gcc \
        git \
        go \
        imagemagick \
        imagemagick-dev \
        imagemagick-heic \
        imagemagick-jpeg \
        imagemagick-webp \
        libheif \
        libc-dev \
        wget

WORKDIR $GOPATH/src/github.com/autowp/goautowp/

# Populate the module cache based on the go.{mod,sum} files.
COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify

COPY . $GOPATH/src/github.com/autowp/goautowp/

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /goautowp cmd/goautowp/goautowp.go

###########################
FROM docker.io/library/alpine:edge

LABEL app-name="autowp.goautowp" \
      maintainer="dmitry@pereslegin.ru"

ENV USER=appuser \
    UID=10001

EXPOSE 8080

ENTRYPOINT ["/goautowp"]

WORKDIR /

RUN apk --update --no-cache add \
        aom-libs \
        ca-certificates \
        imagemagick \
        imagemagick-heic \
        imagemagick-jpeg \
        imagemagick-webp \
        libheif \
        tzdata \
    && \
    # See https://stackoverflow.com/a/55757473/12429735RUN
    adduser \
        --disabled-password \
        --gecos "" \
        --home "/nonexistent" \
        --shell "/sbin/nologin" \
        --no-create-home \
        --uid "${UID}" \
        "${USER}"

COPY etc/ImageMagick-7/policy.xml /etc/ImageMagick-7/policy.xml

USER appuser:appuser

COPY --from=builder /goautowp /goautowp
COPY migrations /migrations
COPY postgres-migrations /postgres-migrations
COPY defaults.yaml /defaults.yaml
