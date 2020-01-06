#!/bin/bash

add-apt-repository -y ppa:longsleep/golang-backports
apt-get -q update && apt-get -yqq install golang git

add-apt-repository -y ppa:bitmark/bitmarkd
add-apt-repository -y ppa:bitmark/bitmark-wallet
apt-get -q update && apt-get -yqq install bitmarkd bitmark-cli bitmark-info bitmark-wallet

sed -ie '/add_port("0.0.0.0", 2135)/d' /etc/bitmarkd.conf.sub
sed -ie '/add_port("0.0.0.0", 2136)/d' /etc/bitmarkd.conf.sub
sed -ie '/add_port("0.0.0.0", 2138)/d' /etc/bitmarkd.conf.sub
sed -ie '/add_port("0.0.0.0", 2139)/d' /etc/bitmarkd.conf.sub
sed -ie '/announce_ips\ =\ interface_public_ips/s/^--//g' /etc/bitmarkd.conf

rm -rf go bitmarkd
apt-get -y purge golang git

