#!/bin/bash

add-apt-repository -y ppa:longsleep/golang-backports
apt-get -yqq update && apt-get -yqq install golang

git clone https://github.com/bitmark-inc/bitmarkd
cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
cd

git clone https://github.com/bitmark-inc/bitmark-wallet
cd bitmark-wallet && git checkout v0.6.3 && mkdir bin
go build -o bin -ldflags "-X main.version=0.6.3" ./...
cd

mv bitmarkd/bin/bitmarkd /usr/local/sbin/
mv bitmarkd/bin/recorderd /usr/local/sbin/

mv bitmarkd/bin/* /usr/local/bin/
mv bitmark-wallet/bin/* /usr/local/bin/

rm -rf go bitmarkd
apt-get -y purge golang
