FROM golang:1.12 as builder

ARG ACTIONS_VERSION

ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /go/src/github.com/variantdev/go-actions
COPY . /go/src/github.com/variantdev/go-actions

RUN if [ -n "${ACTIONS_VERSION}" ]; then git checkout -b tag refs/tags/${ACTIONS_VERSION} || git checkout -b branch ${ACTIONS_VERSION}; fi \
    && make build -e GO111MODULE=on

FROM buildpack-deps:scm

COPY --from=builder /go/src/github.com/variantdev/go-actions/bin/actions /usr/local/bin/actions

ENTRYPOINT ["/usr/local/bin/actions"]
CMD ["--help"]
