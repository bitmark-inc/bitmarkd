# SPDX-License-Identifier: ISC
# Copyright (c) 2019-2024 Bitmark Inc.
# Use of this source code is governed by an ISC
# license that can be found in the LICENSE file.


ARCH = $(shell /usr/bin/uname -m)

VERSION = $(shell \
  changelog="debian/changelog" ; \
  [ -f "$${changelog}" ] && version=$$(head -n 1 "$${changelog}" | sed -E 's/^[^(]*[(]([^)]+)-[[:digit:][:alpha:]]+[)].*$$/\1/') ; \
  git_head="$$(git rev-list --max-count=1 HEAD)" ; \
  git_tag="$$(git rev-list --max-count=1 "v$${version}" || true)" ; \
  git_n="$$(git rev-list --count "v$${version}..HEAD" || echo 0)" ; \
  git_hash_prefix="$${git_head#????????}" ; \
  git_hash_prefix="$${git_head%$${git_hash_prefix}}" ; \
  [ X"$${git_head}" != X"$${git_tag}" ] && version="$${version}+$${git_n}-$${git_hash_prefix}" ; \
  printf '%s' "$${version}")


.PHONY: all
all: install

.PHONY: version
version:
	printf 'version: %s\n' '${VERSION}'


# bitmarkd and command
.PHONY: install
install:
	go install -v -ldflags="-w -s -X main.version=${VERSION}" -gcflags='-e' ./command/...

# local bitmarkd
.PHONY: build
build:
	for p in ./command/* ; \
	do \
	  base="$${p##*/}" ; \
	  printf '===> compiling: %-20s  version: \033[1;34m%s\033[0m\n' "$${base}" "${VERSION}" ; \
	  go build -o "./bin/$${base}" -v -ldflags="-w -s -X main.version=${VERSION}" -gcflags='-e' "./command/$${base}" ; \
	done


# to do a `go get -u` and not fail on private repos
# ensure that ${XDG_CONFIG_HOME}/git/config contains:
# [url "ssh://git@github.com/bitmark-inc/"]
#     insteadof = https://github.com/bitmark-inc/
.PHONY: update-deps
update-deps:
	go mod tidy
	env GOPRIVATE='github.com/bitmark-inc/' go get -u ./...
	go mod tidy

PORT_TUPLE = port-tuple.mk

# for FreeBSD ports
.PHONY: tuple
tuple:
	rm -f "${PORT_TUPLE}"
	rm -rf vendor/
	go mod vendor
	modules2tuple vendor/modules.txt > "${PORT_TUPLE}"
	@printf '\n===> tuple data written to: %s\n\n' "${PORT_TUPLE}"
	rm -r vendor/


# TESTING

.PHONY: test
test:
	go test -v ./...

.PHONY: vet
vet:
	go mod tidy
	@-[ X"$$(uname -s)" = X"FreeBSD" ] && set-sockio
	go vet -v ./... 2>&1 | \
	  awk '/^#.*$$/{ printf "\033[31m%s\033[0m\n",$$0 } /^[^#]/{ print $$0 }'

.PHONY: clean
clean:
	rm -rf bin

.PHONY: get-gocritic
get-gocritic:
	go install -v github.com/go-critic/go-critic/cmd/gocritic@latest

CR_DISABLED = paramTypeCombine
CR_DISABLED += unnamedResult
CR_DISABLED += unlabelStmt
CR_DISABLED += commentFormatting
CR_DISABLED += commentedOutCode
CR_DISABLED += ifElseChain
CR_DISABLED += sloppyTestFuncName
CR_DISABLED += hugeParam

CR_DISABLED += redundantSprint
CR_DISABLED += commentedOutImport

SP = ${EMPTY} ${EMPTY}
CM = ${EMPTY},${EMPTY}
DIS = $(subst ${SP},${CM},${CR_DISABLED})

.PRONY: critic
critic:
	gocritic check -enableAll -enable='#opinionated' -disable="${DIS}" ./...


# Makefile debugging

# use like make print-VARIABLE_NAME
# note "print-" is always lowercase, VARIABLE_NAME is case sensitive
.PHONY: print-%
print-%:
	@printf '%s: %s\n' "$(patsubst print-%,%,${@})"  "${$(patsubst print-%,%,${@})}"
