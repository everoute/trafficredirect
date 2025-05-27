FROM registry.smtx.io/sdn-base/golang:1.20 as builder

RUN git config --global url."ssh://git@github.com".insteadOf "https://github.com"
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts

RUN mkdir -p $GOPATH/src/trafficredirect
WORKDIR $GOPATH/src/trafficredirect

ADD . $GOPATH/src/trafficredirect
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=ssh go mod download

ENV CGO_ENABLED=0
ARG RELEASE_VERSION
ARG GIT_COMMIT
ARG PRODUCT_NAME

RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod --mount=type=ssh \
    go build -o bin/tr-controller -ldflags " \
    -X github.com/everoute/version.releaseVersion=${RELEASE_VERSION} \
    -X github.com/everoute/version.releaseCommit=${GIT_COMMIT} \
    -X github.com/everoute/version.productName=${PRODUCT_NAME}" cmd/controller/*.go

FROM registry.smtx.io/sdn-base/alpine:3.13.6

ARG RELEASE_VERSION
ENV RELEASE_VERSION=${RELEASE_VERSION}
ARG GIT_COMMIT
ENV GIT_COMMIT=${GIT_COMMIT}

RUN mkdir -p /opt/everoute/bin
COPY --from=builder /go/src/trafficredirect/bin/* /opt/everoute/bin/
