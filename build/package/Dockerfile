FROM ubuntu:jammy as builder

ENV DEBIAN_FRONTEND=non-interactive \
    PATH=$PATH:/usr/local/go/bin:$HOME/go/bin \
    GOPATH=$HOME/go

RUN apt-get update -qqy && \
    apt-get dist-upgrade -qqy && \
    apt-get install -qqy \
        ca-certificates \
        gcc \
        git \
        pkg-config \
        gcc \
        git \
        imagemagick \
        libmagickwand-dev \
        libc-dev \
        wget \
    && \
    update-ca-certificates && \
    wget -q https://go.dev/dl/go1.18.3.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.3.linux-amd64.tar.gz && \
    rm go1.18.3.linux-amd64.tar.gz

WORKDIR $GOPATH/src/github.com/autowp/goautowp/

# Populate the module cache based on the go.{mod,sum} files.
COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify

COPY . $GOPATH/src/github.com/autowp/goautowp/

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app cmd/goautowp/goautowp.go

############################
FROM ubuntu:jammy

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
        xmlstarlet \
    && \
    update-ca-certificates && \
    cat /etc/ImageMagick-6/policy.xml | \
            xmlstarlet ed -u "/policymap/policy[@domain='resource'][@name='memory']/@value" -v "2GiB" | \
            xmlstarlet ed -u "/policymap/policy[@domain='resource'][@name='disk']/@value" -v "10GiB" > /etc/ImageMagick-6/policy2.xml \
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

USER appuser:appuser

COPY --from=builder /app /app
COPY migrations /migrations
COPY traffic-migrations /traffic-migrations
COPY defaults.yaml /defaults.yaml
