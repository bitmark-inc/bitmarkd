# Release Notes

Version: 0.12.4

# How to Upgrade

If upgrading from an older setup:

* create a new `bitmarkd.conf` based on data from old configuration
  and the template from the current install.  The file should be self
  explanetarty, but there are some notes in the **Configuration
  Options** section below.
* install the `bitmarkd.conf.sub` in same direcory as `bitmarkd.conf`
* clean up the `/var/lib/bitmarkd` working directory by:
  + remove any json or cache files from the root of the data directory
    as these will now be places in
  + remove and blocks and index LevelDB files in the `data`
    subdirectory

The above actions will re-download the entire blockchain for the new format LevelDB


# Downgrade warning

If the blocks and index LevelDBs have been removed then a downgrade
will have to reload the block chain.


# Notable changes

* Fix the `peers.json` corruption by relocating each blockchain's
  version to its own subdirectory
* Fix memory leak caused by Litecoin connection trying to use Bitcoin
  peers
* LevelDB layout improvements and fast download of block chain

## Data

The naming and location of data and cache files has been changed

* all cache files are moved to deparate subdirectories for each blockchain
  + Bitcoin `peers.json`
  + Litecoin `peers.json`
  + bitmarkd `peers.json` and `reservoir.cache`
* the `blocks` and `index` LevelDBs have been merged to a single
  `bitmark` LevelDB for greater resilience

## RPCs

No RPC have changed

## Configuration options

Quick configuration is now split out from the main file and the
standard `bitmarkd.conf` uses `bitmarkd.conf.sub` to perform the
configuration setup.

Just fill out a minimum set of global values in `bitmarkd.conf`
including these values:

* the chain, normally "bitmark"
* payment addresses for Bitcoin and Litecoin
* a list of announce IP addresss, one each oof IPv4 and IPv6.

The payment mode is set to **p2p** by default but can be overridden by defining the `payment_mode` variable.  The following modes are supported:

* **p2p** connect to a small subset of Bitcoin/Litecoin peers to
  receive blocks directly.  This has redundant connections and is the
  default setting.
* **rest** connect to a local bitcoind/litecoind rest interface to
  poll for blocks this should only be over a lan to avoid connection problems as there is no redundacy.
* **noverify** do not verify payments, this is suitable for a query-only node.
* **discovery** connect to discovery proxy.  **This option will be removed in a future version.**


# Change log

See the `debian/changelog` file for a complete list.
