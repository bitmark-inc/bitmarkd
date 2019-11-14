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
sudo cp $HOME/bitmarkd/bin/* /usr/local/sbin/
sudo rm -rf $HOME/go $HOME/bitmarkd
sudo pkg remove -y perl5 go
sudo pkg clean -ay

# https://github.com/terraform-providers/terraform-provider-digitalocean/issues/243#issuecomment-508226846
cat /dev/null > $HOME/.ssh/authorized_keys
sudo rm -rf /var/lib/cloud/instance
