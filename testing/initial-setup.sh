#!/bin/sh
# generate all LOCAL bitmarkd configuration configurations

all='1 2 3 4 5 6 7 8 9'
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

CHECK_PROGRAM() {
  local p x
  for p in "$@"
  do
    printf '%-32s ' "${p}"
    x=$(which "${p}")
    if [ $? -ne 0 ]
    then
      printf 'is not on the path\n'
      ok=no
    elif [ ! -x "${x}" ]
    then
      printf 'is not executable\n'
      ok=no
    else
      printf '*OK*\n'
    fi
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
CHECK_PROGRAM bitcoind bitcoin-cli litecoind litecoin-cli jq lua52 genbtcltc
CHECK_PROGRAM restart-all-bitmarkds run-recorderd bm-tester
CHECK_PROGRAM generate-bitmarkd-configuration run-bitcoin run-bitmarkd
CHECK_PROGRAM make-blockchain run-discovery node-info
CHECK_PROGRAM run-litecoin

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
