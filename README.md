# bitmarkd - Main program

To compile simply:

~~~~~
go get github.com/bitmark-inc/bitmarkd
go install -v github.com/bitmark-inc/bitmarkd
~~~~~

# Set up

Create the configuration directory, copy sample configuration, edit it to
set up IPs, ports and local bitcoin testnet connection.

~~~~~
mkdir -p ~/.config/bitmarkd
cp bitmarkd.conf-sample  ~/.config/bitmarkd/bitmarkd.conf
${EDITDOR}   ~/.config/bitmarkd/bitmarkd.conf
~~~~~

Generate key files and certificates.

~~~~~
bitmarkd generate-identity
bitmarkd generate-rpc-cert
bitmarkd generate-mine-cert
~~~~~

Start the program.

~~~~~
bitmarkd
~~~~~
