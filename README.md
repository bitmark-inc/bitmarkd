#bitmark-cli
command line interface for issue, transfer bitmark

```
usage: bitmark-cli [options] <command> [params]
       --help               -h            print this message
       --verbose            -v            verbose result
       --config=DIR         -c DIR       *bitmark-cli config folder
       --identity=NAME      -i NAME       identity name [bitmark-identity]

command params: (* = required)
  setup                                   initialise bitmark-cli configuration
       --network=NET        -n NET        bitmark|testing. Connect to which bitmark network [testing]
       --connect=HOST:PORT  -x HOST:PORT *bitmarkd host/IP and port
       --description=TEXT   -d TEXT      *identity description

  generate                                new identity
       --description=TEXT   -d TEXT      *identity description

  issue                                   create and issue bitmark
       --asset=NAME         -a NAME      *asset name
       --description=TEXT   -d TEXT      *asset description
       --fingerprint=TEXT   -f TEXT      *asset fingerprint
       --quantity=N         -q N          quantity to issue [1]

  transfer                                transfer bitmark
       --txid=HEX           -t HEX       *transaction id to transfer
       --receiver=NAME      -r NAME      *identity name to receive the transactoin

  info                                    display bitmarkd status

  version                                 display bitmark-cli version
```
