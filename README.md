# bitmarkd - Main program

[![Made by](https://img.shields.io/badge/Made%20by-Bitmark%20Inc-lightgrey.svg)](https://bitmark.com)
[![GoDoc](https://godoc.org/github.com/bitmark-inc/bitmarkd?status.svg)](https://godoc.org/github.com/bitmark-inc/bitmarkd)
[![Go Report Card](https://goreportcard.com/badge/github.com/bitmark-inc/bitmarkd)](https://goreportcard.com/report/github.com/bitmark-inc/bitmarkd)

Prerequisites

* Install the go language package for your system
* Configure environment variables for go system
* Install the ZMQ4 and Argon2 libraries


## FreeBSD

~~~~~
pkg install libzmq4 libargon2
~~~~~

## MacOSX

(be sure that homebrew is installed correctly)
~~~~
brew tap bitmark-inc/bitmark
brew install argon2
brew install zeromq43
~~~~

## Ubuntu
(tested on distribution 18.04)

Install following packages
   `sudo apt install libargon2-0-dev uuid-dev libzmq3-dev`

To compile simply:

~~~~~
go get github.com/bitmark-inc/bitmarkd
go install -v github.com/bitmark-inc/bitmarkd/command/bitmarkd
~~~~~

:warning: **Argon2 optimization**

Argon2 can achieve better performance if [AVX instructions](https://en.wikipedia.org/wiki/Advanced_Vector_Extensions) is available. But the potential optimization is not enabled if Arong2 is intalled by package managers.

To leverage AVX instructions, extra flag has to be specified during the compilation process.

```shell
make OPTTARGET=native
```

If AVX is not available, make sure Arong2 has no reference to AVX otherwise bitmarkd will crash. 

```shell
make OPTTARGET=generic
```

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

* setup git hooks

  Link git hooks directory, run command `./scripts/setup-hook.sh` at root of bitmarkd
  directory. Currently it provides checkings for two stages:

  1. Before commit (`pre-commt`)

	Runs `go lint` for every modified file(s). It shows suggestions but not
    necessary to follow.

  2. Before push to remote (`pre-push`)

  	Runs `go test` for whole directory except `vendor` one. It is
    mandatory to pass this check because generally, new modifications should not
    break existing logic/behavior.

    Other optional actions are `sonaqube` and `go tool vet`. These two are
    optional to follow since static code analysis just provide some advice.

* all variables are camel case i.e. no underscores
* labels are all lowercase with '_' between words
* imports and one single block
* all break/continue must have label
* avoid break in switch and select
