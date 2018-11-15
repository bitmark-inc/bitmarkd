# bitmarkd - Main program

[![GoDoc](https://godoc.org/github.com/bitmark-inc/bitmarkd?status.svg)](https://godoc.org/github.com/bitmark-inc/bitmarkd)

Prerequisites

* Install the go language package for your system
* Configure environment variables for go system
* install the ZMQ4, UCL, Argon2 libraries

For shell add the following to the shell's profile
(remark the `export CC=clang` if you wish to use gcc)
~~~~~
# check for go installation
GOPATH="${HOME}/gocode"
if [ -d "${GOPATH}" ]
then
  gobin="${GOPATH}/bin"
  export GOPATH
  export PATH="${PATH}:${gobin}"
  # needed for FreeBSD 10 and later
  export CC=clang
else
  unset GOPATH
fi
unset gobin
~~~~~

On FreeBSD/PC-BSD

~~~~~
pkg install libzmq4 libargon2 libucl
~~~~~

On a Debian like system
(as of Ubuntu 14.04 this only has V3, so need to search for PPA)

~~~~~
apt-get install libzmq4-dev
# lib ucl and argon2 need to be manually installed
# see the Makefile in c-libraries to set up local static copies
~~~~~

On a macosx
(be sure that homebrew is installed correctly)
~~~~
brew install libucl
brew install argon2

brew tap bitmark-inc/bitmark
brew install zeromq41
~~~~

On Ubuntu
(tested dor distribution 18.04)

1. Install following packages
   `sudo apt install libargon2-0-dev autoconf libtool pkg-config uuid-dev libzmq3-dev`

2. Compile and install ucl

    ```
    git clone https://github.com/vstakhov/libucl.git
    cd libucl
    ./autogen.sh
    ./configure --enable-utils
    make
    make install
    ```

To compile simply:

~~~~~
go get github.com/bitmark-inc/go-libucl
go get github.com/bitmark-inc/go-argon2
go get github.com/bitmark-inc/bitmarkd
go install -v github.com/bitmark-inc/bitmarkd/command/bitmarkd
~~~~~

# Set up

Create the configuration directory, copy sample configuration, edit it to
set up IPs, ports and local bitcoin testnet connection.

~~~~~
mkdir -p ~/.config/bitmarkd
cp command/bitmarkd/bitmarkd.conf.sample  ~/.config/bitmarkd/bitmarkd.conf
${EDITOR}   ~/.config/bitmarkd/bitmarkd.conf
~~~~~

To see the bitmarkd sub-commands:

~~~~~
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" help
~~~~~

Generate key files and certificates.

~~~~~
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" gen-peer-identity
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" gen-rpc-cert
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" gen-proof-identity
~~~~~

Start the program.

~~~~~
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" start
~~~~~

Note that a similar process is needed for the prooferd (mining subsystem)

# Prebuilt Binary

* Flatpak

    Please refer to [wiki](https://github.com/bitmark-inc/bitmarkd/wiki/Instruction-for-Flatpak-Prebuilt)

* Docker

    Please refer to [bitmark-node](https://github.com/bitmark-inc/bitmark-node)

# Coding

* all variables are camel case i.e. no underscores
* labels are all lowercase with '_' between words
* imports and one single block
* all break/continue must have label
* avoid break in switch and select
