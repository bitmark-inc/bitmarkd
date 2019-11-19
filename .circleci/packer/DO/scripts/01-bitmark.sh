#!/bin/bash

add-apt-repository -y ppa:longsleep/golang-backports
apt-get -yqq update && apt-get -yqq install golang

git clone https://github.com/bitmark-inc/bitmarkd
cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
cd
cp bitmarkd/bin/* /usr/local/sbin/
rm -rf go bitmarkd
apt-get -y purge golang
