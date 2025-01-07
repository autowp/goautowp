FROM docker.io/library/ubuntu:noble AS builder

ENV DEBIAN_FRONTEND=non-interactive \
    PATH=$PATH:/usr/local/go/bin:$HOME/go/bin \
    GOPATH=$HOME/go

RUN apt-get update -qqy && \
    apt-get dist-upgrade -qqy && \
    apt-get install -qqy \
        ca-certificates \
        gcc \
        git \
        imagemagick \
        libc-dev \
        libmagickwand-dev \
        pkg-config \
        wget \
    && \
    apt-get clean && \
    update-ca-certificates && \
    wget -q https://go.dev/dl/go1.23.4.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz && \
    rm go1.23.4.linux-amd64.tar.gz

WORKDIR $GOPATH/src/github.com/autowp/goautowp/

# Populate the module cache based on the go.{mod,sum} files.
COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify

COPY . $GOPATH/src/github.com/autowp/goautowp/

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app cmd/goautowp/goautowp.go

############################
FROM docker.io/library/ubuntu:noble

LABEL app_name="autowp.goautowp" \
      maintainer="dmitry@pereslegin.ru"

ENV DEBIAN_FRONTEND=non-interactive \
    USER=appuser \
    UID=10001

EXPOSE 8080

ENTRYPOINT ["/app"]

WORKDIR /

RUN apt-get update -qqy && \
    apt-get dist-upgrade -qqy && \
    apt-get install -qqy \
        ca-certificates \
        imagemagick \
        tzdata \
    && \
    apt-get clean && \
    update-ca-certificates && \
    # See https://stackoverflow.com/a/55757473/12429735RUN
    adduser \
        --disabled-password \
        --gecos "" \
        --home "/nonexistent" \
        --shell "/sbin/nologin" \
        --no-create-home \
        --uid "${UID}" \
        "${USER}"

COPY etc/ImageMagick-6/policy.xml /etc/ImageMagick-6/policy.xml

USER appuser:appuser

COPY --from=builder /app /app
COPY migrations /migrations
COPY postgres-migrations /postgres-migrations
COPY defaults.yaml /defaults.yaml
