# bitmarkd - Main program

[![Made by](https://img.shields.io/badge/Made%20by-Bitmark%20Inc-lightgrey.svg)](https://bitmark.com)
[![GoDoc](https://godoc.org/github.com/bitmark-inc/bitmarkd?status.svg)](https://godoc.org/github.com/bitmark-inc/bitmarkd)
[![Go Report Card](https://goreportcard.com/badge/github.com/bitmark-inc/bitmarkd)](https://goreportcard.com/report/github.com/bitmark-inc/bitmarkd)
[![CircleCI](https://circleci.com/gh/bitmark-inc/bitmarkd.svg?style=svg)](https://circleci.com/gh/bitmark-inc/bitmarkd)

Prerequisites

* Install the go language package for the system
* Configure environment variables for go system
* Install the ZMQ4 and Argon2 libraries

# Operating system specific setup commands

## FreeBSD

~~~~~
pkg install libzmq4 libargon2 git
~~~~~

## MacOSX

(be sure that homebrew is installed correctly)
~~~
brew install argon2
brew install zeromq
brew install git
~~~

## Ubuntu
(tested on version 18.04)

Install following packages

~~~
sudo apt install libargon2-0-dev uuid-dev libzmq3-dev git
~~~

## Debian
(tested on version 9)

First we need to add access to testing package's repository as well as
to our current version, in this case stable.

~~~
root@debian-bitmarkd:/# cat /etc/apt/sources.list.d/stable.list
deb     http://ftp.de.debian.org/debian/    stable main contrib non-free
deb-src http://ftp.de.debian.org/debian/    stable main contrib non-free
deb     http://security.debian.org/         stable/updates  main contrib non-free

root@debian-bitmarkd:/# cat /etc/apt/sources.list.d/testing.list
deb     http://ftp.de.debian.org/debian/    testing main contrib non-free
deb-src http://ftp.de.debian.org/debian/    testing main contrib non-free
deb     http://security.debian.org/         testing/updates  main contrib non-free
~~~

Now install libargon2 using:
```
apt-get -t testing install libargon2-dev libargon2-1
```

For the other packages, install from stable or testing, both versions work:
```
apt install uuid-dev libzmq3-dev
apt install git
```

# Compilation commands for all operating systems

To compile use use the `git` command to clone the repository and the
`go` command to compile all commands.  The process requires that the
Go installation be 1.12 or later as the build process uses Go Modules.

~~~~~
git clone https://github.com/bitmark-inc/bitmarkd
cd bitmarkd
go install -v ./...
~~~~~

# Set up for running a node

Note: ensure that the `${HOME}/go/bin` directory is on the path before
continuing.  The commands below assume that a checked out and compiled
version of the system exists in the `${HOME}/bitmarkd` directory.

## Setup and run bitmarkd

Create the configuration directory, copy sample configuration, edit it
to set up IP addresses, ports and local bitcoin testnet connection.
The sample configuration has some embedded instructions for quick
setup and only a few items near the beginning of the file need to be
set for basic use.

~~~~~
mkdir -p ~/.config/bitmarkd
cp ~/bitmarkd/command/bitmarkd/bitmarkd.conf.sample  ~/.config/bitmarkd/bitmarkd.conf
${EDITOR} ~/.config/bitmarkd/bitmarkd.conf
~~~~~

To see the bitmarkd sub-commands:

~~~~~
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" help
~~~~~

Generate key files and certificates.

~~~~~
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" gen-peer-identity "${HOME}/.config/bitmarkd/"
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" gen-rpc-cert "${HOME}/.config/bitmarkd/"
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" gen-proof-identity "${HOME}/.config/bitmarkd/"
~~~~~

Start the program.

~~~~~
bitmarkd --config-file="${HOME}/.config/bitmarkd/bitmarkd.conf" start
~~~~~


## Setup and run recorderd (the mining program)

This is similar to the bitmarkd steps above. For mining on the local
bitmarkd the sample configuration should work without changes.

~~~~~
mkdir -p ~/.config/recorderd
cp ~/bitmarkd/command/recorderd/recorderd.conf.sample  ~/.config/recorderd/recorderd.conf
${EDITOR} ~/.config/recorderd/recorderd.conf
~~~~~

To see the recorderd sub-commands:

~~~~~
recorderd --config-file="${HOME}/.config/recorderd/recorderd.conf" help
~~~~~

Generate key files and certificates.

~~~~~
recorderd --config-file="${HOME}/.config/recorderd/recorderd.conf" generate-identity "${HOME}/.config/recorderd/
~~~~~

Start the program.

~~~~~
recorderd --config-file="${HOME}/.config/recorderd/recorderd.conf" start
~~~~~


## Setup and run bitmark-cli (command line program to send transactions)

Initialise the bitmark-cli for live and test networks creating a new
account on each network.  This setup is presuming that live or test
bitmarkd will be running on the same machine so thatthe loopback
address can be used.

~~~~~
mkdir -p ~/.config/bitmark-cli
bitmark-cli -n bitmark -i mylive setup -c 127.0.0.1:2130  -d 'my first live account' -n
bitmark-cli -n testing -i mytest setup -c 127.0.0.1:12130  -d 'my first testing account' -n
~~~~~

To see the cli sub-commands: or details on a specific command e.g., "setup"

~~~~~
bitmark-cli help
bitmark-cli help setup
~~~~~

Note: If you wish to add more connections or change the default
connection created above then it is necessary to edit the JSON
configuration files to modify the connections list.


# Prebuilt Binary

* FreeBSD

**Install bitmarkd bitmark-cli and recorderd**
~~~~~
pkg install bitmark
~~~~~

**Alternatively select one or more individual packages**
~~~~~
pkg install bitmark-daemon
pkg install bitmark-recorder
pkg install bitmark-cli
~~~~~

* Flatpak

    Please refer to [wiki](https://github.com/bitmark-inc/bitmarkd/wiki/Instruction-for-Flatpak-Prebuilt)

* Docker

    Please refer to [bitmark-node](https://github.com/bitmark-inc/bitmark-node)

# Coding

* setup git hooks

  Link git hooks directory, run command `./scripts/setup-hook.sh` at root of bitmarkd
  directory. Currently it provides checks for two stages:

  1. Before commit (`pre-commit`)

	Runs `go lint` for every modified file. It shows suggestions but not
    necessary to follow.

  2. Before push to remote (`pre-push`)

    Runs `go test` for whole directory. It is mandatory to pass this
    check because generally, new modifications must not break existing
    logic/behaviour.

    Other optional actions are `sonaqube` and `go tool vet`. These two are
    optional to follow since static code analysis just provide some advice.

* all variables are camel case with no underscores
* labels are all lowercase with '_' between words
* imports and one single block
* all break/continue must have label
* avoid break in switch and select
