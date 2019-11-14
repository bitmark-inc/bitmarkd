#!/bin/sh

if [ "$PACKER_BUILDER_TYPE" != "amazon-ebs" ]
then
    return 0;
fi

su root -c 'pkg install --yes pkgconf-1.6.3,1 git go'
su root -c 'pkg install --yes libargon2-20190702 libzmq4-4.3.1_1'

git clone https://github.com/bitmark-inc/bitmarkd
cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
su root -c "cp $HOME/bitmarkd/bin/* /usr/local/sbin/"
su root -c "rm -rf $HOME/go $HOME/bitmarkd"
su root -c 'pkg remove -y perl5 go'
su root -c 'pkg clean -ay'

rm -f $HOME/.ssh/authorized_keys
su root -c 'touch /firstboot'
