#!/bin/sh

sudo pkg install --yes pkgconf-1.6.3,1 git go
sudo pkg install --yes libargon2-20190702 libzmq4-4.3.1_1

git clone https://github.com/bitmark-inc/bitmarkd
cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
cd

git clone https://github.com/bitmark-inc/bitmark-wallet
cd bitmark-wallet && git checkout v0.6.3 && mkdir bin
go build -o bin -ldflags "-X main.version=0.6.3" ./...
cd

sudo mv bitmarkd/bin/bitmarkd /usr/local/sbin/
sudo mv bitmarkd/bin/recorderd /usr/local/sbin/

sudo mv bitmarkd/bin/* /usr/local/bin/
sudo mv bitmark-wallet/bin/* /usr/local/bin/

rm -rf go bitmarkd
rm -f .ssh/authorized_keys

sudo rm -f /etc/ssh/*key*
sudo pkg remove -y perl5 go
sudo pkg clean -ay

sudo rm -rf /tmp/* /var/tmp/*
sudo rm -rf /var/lib/cloud/instances/*
sudo rm -rf /var/lib/cloud/instance

sudo rm -rf /var/log/*.gz /var/log/*.[0-9] /var/log/*-???????? /var/log/*.log /var/log/utx.*
sudo dd if=/dev/zero of=/zerofile && sync; sudo rm /zerofile; sync
