build/pullvet:
	go build -o bin/pullvet ./cmd/pullvet

build/exec:
	go build -o bin/exec ./cmd/exec

build:
	go build -o bin/actions ./cmd

install: build
	mv bin/actions ~/bin/actions

test/integration:
	echo aaa
	GITHUB_EVENT_NAME=issues GITHUB_EVENT_PATH=testdata/issues_event.json bin/actions exec -status-context milestone -- bin/actions pullvet -require-any -milestone-match 'test-v.+' label milestone/none

test:
	go test ./...

# Run this like: SOURCE_BRANCH= make build/docker so that you can build the `latest` from a commit that is not tagged yet
build/docker: SOURCE_BRANCH ?= v1.2.3
build/docker:
	SOURCE_BRANCH=$(SOURCE_BRANCH) DOCKERFILE_PATH=./Dockerfile IMAGE_NAME=variantdev/actions ./hooks/build

run/docker:
	docker run --rm variantdev/actions:latest

release/minor:
	git checkout master
	git pull --rebase origin master
	bash -c 'if git branch | grep autorelease; then git branch -D autorelease; else echo no branch to be cleaned; fi'
	git checkout -b autorelease origin/master
	bash -c 'SEMTAG_REMOTE=origin hack/semtag final -s minor'
	git checkout master

release/patch:
	git checkout master
	git pull --rebase origin master
	bash -c 'if git branch | grep autorelease; then git branch -D autorelease; else echo no branch to be cleaned; fi'
	git checkout -b autorelease origin/master
	bash -c 'SEMTAG_REMOTE=origin hack/semtag final -s patch'
	git checkout master
