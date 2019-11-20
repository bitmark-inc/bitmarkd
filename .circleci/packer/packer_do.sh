#!/bin/sh

if [ "$PACKER_BUILDER_TYPE" != "digitalocean" ]
then
    return 0;
fi

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

su root -c "mv bitmarkd/bin/bitmarkd /usr/local/sbin/"
su root -c "mv bitmarkd/bin/recorderd /usr/local/sbin/"

su root -c "mv bitmarkd/bin/* /usr/local/bin/"
su root -c "mv bitmark-wallet/bin/* /usr/local/bin/"

sudo rm -rf $HOME/go $HOME/bitmarkd
sudo pkg remove -y perl5 go
sudo pkg clean -ay

# https://github.com/terraform-providers/terraform-provider-digitalocean/issues/243#issuecomment-508226846
cat /dev/null > $HOME/.ssh/authorized_keys
sudo rm -rf /var/lib/cloud/instance
