#!/bin/sh

unset HISTFILE

apt-get -q update
apt-get -yq install software-properties-common

add-apt-repository -y ppa:longsleep/golang-backports
apt-get -yqq update && apt-get -yqq install libargon2-0-dev libzmq3-dev golang git

git clone https://github.com/bitmark-inc/bitmarkd
cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
cd
cp bitmarkd/bin/* /usr/local/sbin/
rm -rf go bitmarkd
apt-get -y purge git golang
