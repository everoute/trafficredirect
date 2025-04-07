.PHONY: image-generate generate docker-generate test docker-test publish

CONTROLLER_GEN=$(shell which controller-gen)

image-generate:
	docker build -f build/image/generate/Dockerfile -t api/generate ./build/image/generate/

generate: manifests codegen

manifests:
	$(CONTROLLER_GEN) crd paths="./servicechain/..." output:crd:dir=deploy/servicechain/crds output:stdout

codegen:
	deepcopy-gen -O zz_generated.deepcopy --go-header-file ./hack/boilerplate.generatego.txt --input-dirs ./servicechain/...

docker-generate: image-generate
	$(eval WORKDIR := /go/src/github.com/everoute/api)
	docker run --rm -iu 0:0 -w $(WORKDIR) -v $(CURDIR):$(WORKDIR) api/generate make generate

test:
	go test ./... --race --coverprofile coverage.out

docker-test:
	$(eval WORKDIR := /go/src/github.com/everoute/api)
	docker run --rm -iu 0:0 -w $(WORKDIR) -v $(CURDIR):$(WORKDIR) golang:1.19 make test

publish:
