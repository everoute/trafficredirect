.PHONY: image-generate generate docker-generate test docker-test publish

CONTROLLER_GEN=$(shell which controller-gen)

image:
	docker build -f build/image/release/Dockerfile -t registry.smtx.io/everoute/tr-controller . --build-arg RELEASE_VERSION="v0.0.0" --build-arg GIT_COMMIT="local" --build-arg PRODUCT_NAME="everoute"

image-generate:
	docker build -f build/image/generate/Dockerfile -t tr/generate ./build/image/generate/

generate: manifests codegen

manifests:
	$(CONTROLLER_GEN) crd paths="./api/..." output:crd:dir=deploy/chart/templates/crds output:stdout

codegen:
	deepcopy-gen -O zz_generated.deepcopy --go-header-file ./hack/boilerplate.generatego.txt --input-dirs=./api/trafficredirect/v1alpha1,./api/trafficredirect

docker-generate: image-generate
	$(eval WORKDIR := /go/src/github.com/everoute/trafficredirect)
	docker run --rm -iu 0:0 -w $(WORKDIR) -v $(CURDIR):$(WORKDIR) tr/generate make generate

test:
	go test ./... --race --coverprofile coverage.out

docker-test:
	$(eval WORKDIR := /go/src/github.com/everoute/trafficredirect)
	docker run --rm -iu 0:0 -w $(WORKDIR) -v $(CURDIR):$(WORKDIR) golang:1.20 make test

publish:
