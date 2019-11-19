#!/bin/sh

su root -c 'pkg install --yes pkgconf-1.6.3,1 git go'
su root -c 'pkg install --yes libargon2-20190702 libzmq4-4.3.1_1'

git clone https://github.com/bitmark-inc/bitmarkd
cd bitmarkd && git checkout v$BITMARKD_VERSION && mkdir bin
go build -o bin -ldflags "-X main.version=$BITMARKD_VERSION" ./...
cd

su root -c "cp bitmarkd/bin/* /usr/local/sbin/"
su root -c "rm -rf go bitmarkd"

rm -f .ssh/authorized_keys
su root -c 'rm -f /etc/ssh/*key*'

su root -c 'pkg remove -y perl5 go'
su root -c 'pkg clean -ay'

su root -c 'touch /firstboot'
su root -c 'rm -rf /tmp/* /var/tmp/*'
su root -c 'rm -rf /var/log/*.gz /var/log/*.[0-9] /var/log/*-???????? /var/log/*.log /var/log/utx.*'
su root -c 'dd if=/dev/zero of=/zerofile && sync; rm /zerofile; sync'

su root -c 'cat /dev/null > /var/log/auth.log'
