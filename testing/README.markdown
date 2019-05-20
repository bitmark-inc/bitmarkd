# local test setup

Generate an initial configuration setup:

~~~
sh initial-setup.sh
~~~

Note that this require `XDG_CONFIG_HOME` to be set properly although
it can just use `${HOME}/.config`

At the end are printed some TXT records for the local configuration

Setup a local override using dnsmasq or a local unbound server or even
add them to some real DNS.  Then re-run the configuration with this domain:

~~~
sh initial-setup.sh nodes.somelocal.domain.tld
~~~

That should complete the setup:  to start up run:

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

initial balance with be zero regtest coins so generate 101 blocks

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
