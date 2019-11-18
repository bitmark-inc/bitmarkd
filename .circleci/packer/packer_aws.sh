#!/bin/sh

if [ "$PACKER_BUILDER_TYPE" != "amazon-ebs" ]
then
    return 0;
fi

case $PACKER_BUILD_NAME in
	aws-freebsd)
        su root -c 'pkg install --yes pkgconf-1.6.3,1 git go'
        su root -c 'pkg install --yes libargon2-20190702 libzmq4-4.3.1_1'

        git clone https://github.com/bitmark-inc/bitmarkd
        cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
        go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
        cd $HOME
        su root -c "cp $HOME/bitmarkd/bin/* /usr/local/sbin/"
        su root -c "rm -rf $HOME/go $HOME/bitmarkd"
        su root -c 'pkg remove -y perl5 go'
        su root -c 'pkg clean -ay'

        rm -f $HOME/.ssh/authorized_keys
        su root -c 'touch /firstboot'
		;;
	aws-ubuntu)
        sudo apt-get -q update
        sudo apt-get -yq install software-properties-common

        sudo add-apt-repository -y ppa:longsleep/golang-backports
        sudo apt-get -yqq update && sudo apt-get -yqq install libargon2-0-dev libzmq3-dev golang git

        git clone https://github.com/bitmark-inc/bitmarkd
        cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
        go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
        cd $HOME
        sudo cp $HOME/bitmarkd/bin/* /usr/local/sbin/
        sudo rm -rf $HOME/go $HOME/bitmarkd
        sudo apt-get -y purge git golang
        sudo apt-get -y autoremove
        sudo apt-get -y autoclean

        cat /dev/null > $HOME/.ssh/authorized_keys
        sudo rm -rf /var/lib/cloud/instance/sem/*
		;;
	*)
		echo "Sorry, I don't understand"
		;;
esac
