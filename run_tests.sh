#!/usr/bin/env bash
set -ex

# The script does automatic checking on a Go package and its sub-packages,
# including:
# 1. gofmt         (http://golang.org/cmd/gofmt/)
# 2. go vet        (http://golang.org/cmd/vet)
# 3. gosimple      (https://github.com/dominikh/go-simple)
# 4. unconvert     (https://github.com/mdempsky/unconvert)
# 5. ineffassign   (https://github.com/gordonklaus/ineffassign)
# 6. race detector (http://blog.golang.org/race-detector)

# gometalinter (github.com/alecthomas/gometalinter) is used to run each each
# static checker.

GOVERSION=${1:-1.10}
REPO=lddldata
DOCKER_IMAGE_TAG=Legenddigital-golang-builder-$GOVERSION

testrepo () {
  TMPFILE=$(mktemp)

  # Check lockfile
  dep ensure -no-vendor -dry-run

  # All good, so run for real
  dep ensure

  # Check linters
  gometalinter --vendor --disable-all --deadline=10m \
    --enable=gofmt \
    --enable=vet \
    --enable=gosimple \
    --enable=unconvert \
    --enable=ineffassign \
    ./...
  if [ $? != 0 ]; then
    echo 'gometalinter has some complaints'
    exit 1
  fi

  # Test application install
  if [ $GOVERSION == 1.10 ]; then
    go install -i . ./cmd/...
  else
    go install . ./cmd/...
  fi
  if [ $? != 0 ]; then
    echo 'go install failed'
    exit 1
  fi

  # Check tests
  git clone https://github.com/lddllabs/bug-free-happiness test-data-repo
  tar xvf test-data-repo/stakedb/test_ticket_pool.bdgr.tar.xz

  env GORACE='halt_on_error=1' go test -v -race ./...
  if [ $? != 0 ]; then
    echo 'go tests failed'
    exit 1
  fi

  echo "------------------------------------------"
  echo "Tests completed successfully!"
}

if [ $GOVERSION == "local" ]; then
    testrepo
    exit
fi

docker pull Legenddigital/$DOCKER_IMAGE_TAG
if [ $? != 0 ]; then
        echo 'docker pull failed'
        exit 1
fi

docker run --rm -it -v $(pwd):/src Legenddigital/$DOCKER_IMAGE_TAG /bin/bash -c "\
  rsync -ra --include-from=<(git --git-dir=/src/.git ls-files) \
  --filter=':- .gitignore' \
  /src/ /go/src/github.com/Legenddigital/$REPO/ && \
  cd github.com/Legenddigital/$REPO/ && \
  bash run_tests.sh local"
if [ $? != 0 ]; then
        echo 'docker run failed'
        exit 1
fi
