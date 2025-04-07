FROM golang:1.20

RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.2 && \
    go install k8s.io/code-generator/cmd/deepcopy-gen@kubernetes-1.27.7 && \
    go clean -cache

ENV GOROOT=/usr/local/go
