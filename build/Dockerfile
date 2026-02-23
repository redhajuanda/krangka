# Builder
FROM golang:1.24-alpine as builder
ARG appName
ARG appPath
ARG TARGETOS
ARG TARGETARCH

RUN mkdir /${appName}
COPY . /${appName}
WORKDIR /${appName}
COPY go.mod .
COPY go.sum .
RUN mkdir ~/.ssh/ && \
    apk update; \
    apk upgrade; \
    apk --update add git make gcc musl-dev openssh; \
    chmod 700 ~/.ssh && \
    ssh-keyscan gitlab.sicepat.tech > ~/.ssh/known_hosts && \
    git config --global url."git@gitlab.sicepat.tech:".insteadOf "https://gitlab.sicepat.tech/"
RUN --mount=type=ssh GO111MODULE=on go mod download
# COPY . /${appName}
# RUN go install github.com/swaggo/swag/cmd/swag && swag init
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -a -installsuffix cgo -o ${appName} ${appPath}

# Cert
FROM alpine:latest as certs
RUN apk --update add ca-certificates

# Distribution
FROM alpine:latest
ARG appName
WORKDIR /${appName}

## COPY scuttle binary
## Remove scuttle request from Infra SA
## https://sicepat.slack.com/archives/C02463Y845R/p1716870045086029
## COPY --from=redboxoss/scuttle:1.3.5 /scuttle /bin/scuttle
COPY --from=builder /${appName}/${appName} ./app
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /${appName}/assets ./assets
ENV TZ=Asia/Jakarta
RUN apk add --no-cache tzdata && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone
## WRAP Process with scuttle
## ENTRYPOINT ["scuttle", "./app"]
ENTRYPOINT ["./app"]
