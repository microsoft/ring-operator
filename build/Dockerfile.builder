FROM golang:1.12.7-alpine3.10

RUN apk add git

ENV RELEASE_VERSION v0.9.0

RUN wget -O /usr/local/bin/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && \
    chmod +x /usr/local/bin/operator-sdk