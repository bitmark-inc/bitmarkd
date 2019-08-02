#!/bin/sh
# generate all LOCAL bitmarkd configuration configurations

all=$(seq 1 12)
console='1 2 8'
more='1 2 8'

# to setup the DNS TXT records
dns_txt='1 2'


ERROR() {
  printf 'error: '
  printf "$@"
  printf '\n'
  exit 1
}

SEP() {
  printf '==================================================\n'
  [ -z "${1}" ] && return
  printf '== '
  printf "$@"
  printf '\n'
  printf '==================================================\n'
}

# sets global ok to no if any error
CHECK_PROGRAM() {
  local alt p x i flag
  for alt in "$@"
  do
    flag=no
    i=0
    alt="${alt}:"
    # search alternatives
    while :
    do
      i=$((i + 1))
      p="${alt%%:*}"
      [ -z "${p}" ] && break
      alt="${alt#*:}"
      printf '%2d: %-32s ' "${i}" "${p}"
      x=$(which "${p}")
      if [ $? -ne 0 ]
      then
        printf 'is not on the path\n'
      elif [ ! -x "${x}" ]
      then
        printf 'is not executable\n'
      else
        printf '*OK*\n'
        flag=yes
        break
      fi
    done
    [ X"${flag}" = X"no" ] && ok=no
  done
}


# main program

[ -n "${1}" ] && nodes_domain="${1}"
[ -z "${nodes_domain}" ] && ERROR 'missing nodes-domain argument'

xdg_home="${XDG_CONFIG_HOME}"
[ -z "${xdg_home}" ] && ERROR 'export XDG_CONFIG_HOME="${HOME}/.config"  or similar'
[ -d "${xdg_home}" ] || ERROR 'missing directory: "%s" please create first' "${xdg_home}"

this_dir=$(dirname "$0")
PATH="${this_dir}:${PATH}"
samples="${this_dir}/samples"

# check programs
ok=yes
CHECK_PROGRAM bitmarkd bitmark-cli recorderd discovery bitmark-wallet
CHECK_PROGRAM bitcoind bitcoin-cli
CHECK_PROGRAM litecoind litecoin-cli
CHECK_PROGRAM awk jq lua52:lua5.2:lua53:lua5.3:lua
CHECK_PROGRAM genbtcltc restart-all-bitmarkds bm-tester
CHECK_PROGRAM generate-bitmarkd-configuration
CHECK_PROGRAM run-bitcoin run-litecoin run-discovery
CHECK_PROGRAM run-bitmarkd run-recorderd
CHECK_PROGRAM make-blockchain node-info

# fail if something is missing
[ X"${ok}" = X"no" ] && ERROR 'missing programs'

# check coins setup
for program in bitcoin litecoin discovery recorderd
do
  d="${xdg_home}/${program}"
  mkdir -p "${d}" "${d}/log"
  cf="${program}.conf"
  if [ ! -f "${d}/${cf}" ]
  then
    cp -p "${samples}/${cf}" "${d}/"
    run-${program} --generate
  fi
done

# setup bitmarkd configs
for i in ${console}
do
  eval "console_${i}"=yes
done
for i in ${more}
do
  eval "more_${i}"=yes
done

opts=
OPT() {
  opts="${opts} $*"
}

SEP 'expect errors if here:'

subs=

CONFIGURE() {
  for i in ${all}
  do
    eval console=\"\${console_${i}}\"
    eval more=\"\${more_${i}}\"
    opts=''
    OPT --chain=local
    OPT --bitcoin="${xdg_home}/bitcoin"
    OPT --litecoin="${xdg_home}/litecoin"
    OPT --discovery="${xdg_home}/discovery"
    OPT "$@"
    OPT --update
    [ X"${console}" = X"yes" ] && OPT --console
    [ X"${more}" = X"yes" ] && OPT --more

    generate-bitmarkd-configuration ${opts} "${i}"
    SEP

    public=$(cat "${xdg_home}/bitmarkd${i}/proof.public")
    subs="${subs}s/%%BITMARKD_${i}%%/${public#PUBLIC:}/;"
  done
}

# first pass configure
CONFIGURE

# fixup recorderd
for program in recorderd
do
  d="${xdg_home}/${program}"
  dcf="${d}/${program}.conf"
  [ -f "${dcf}" ] || ERROR 'missing file: %s' "${dcf}"
  sed -E -i .bk "${subs}" "${dcf}"
done

# print out the dns items
SEP 'configure your local DNS TXT records with the following data\n'
for i in ${dns_txt}
do
  run-bitmarkd --config="%${i}" dns-txt
  SEP
done

# add proper nodes and reconfigure
SEP 'update configuration...'
CONFIGURE --nodes="${nodes_domain}" > /dev/null 2>&1
SEP 'finished'
