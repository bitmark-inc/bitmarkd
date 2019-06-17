# Bitmarkd Info

This is a rpc client of bitmarkd in go.

## Usage

Use `bitmark-info -h` to show basic usage of the command

```
$ bitmark-info -h
usage: bitmark-info [--help] [--info-type=TYPE] [host:port]
```

For querying node info, use

```
$ bitmark-info [ip address]:[port]
```

Also, you can select which kind of information you want to query. The
folllowing types are options of information types (TYPE):

1. node (node info, the default value)
2. sbsc (subscriber)
3. conn (connector)

For example, 

```
$ bitmark-info --info-type=sbsc 127.0.0.1:2130
```

It will then shows the connection of subscriber for a bitmarkd.
