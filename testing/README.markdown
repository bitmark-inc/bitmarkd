# local test setup

Generate an initial configuration setup:

Note that this requires `XDG_CONFIG_HOME` to be set properly although
can just use `export XDG_CONFIG_HOME="${HOME}/.config"` if running on
non-desktop system.

At the end are printed some TXT records for the local configuration

Setup a local override using dnsmasq or a local unbound server or even
add them to some real DNS.  Then re-run the configuration with this domain:

run this command:

~~~
initial-setup.sh nodes.somelocal.domain.tld
~~~

That should complete the setup, to start the network up run:

~~~
bm-tester
~~~

This will create a multi-tabbed tmux session with a shell prompt in
the last and currently open tab, run:

~~~
node-info
~~~

And keep running until you get normal nodes (it might take a while the fist time)

# bitcoin and litecoin

initial balance with be zero regtest coins so generate 101 blocks (as
newly minted coins are loked until 100 confirmations have elapsed)

~~~
% run-bitcoin getbalance
0.00000000
% run-litecoin getbalance
0.00000000
% genbtcltc 101
[ lots of output block hashes]
......
% run-bitcoin getbalance
50.00000000
% run-litecoin getbalance
50.00000000
~~~

You can use getnewaddress and other coin commands to make payments
when doing transfers with bitmark-cli


# bitmark-cli

create two users for the make-blockchain script
Create users for all the nodes

~~~
create-node-users.sh
~~~

# setup bitmark-wallet

This will create a wallet (if does not already exist) and add another
25 coins to an existing wallet

~~~
create-bitmark-wallet.sh
~~~


# now create a minimum blockchain

~~~
make-blockchain new btc
~~~

to delete blockchain and regenerate using ltc

~~~
restart-all-bitmarkds -r -p ; make-blockchain new ltc
~~~

Note the old blockchain is actually renamed so could be renamed back
if desired
