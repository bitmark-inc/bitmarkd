#!/bin/sh
# generate all LOCAL bitmarkd configuration configurations

# do not change these defaults (use bm-tester.conf to override)
all=$(seq 1 12) # sets list of daemons to run
console='1 2 8' # sets console=true
more='1 2 8'    # repeat a number to increase detail
recorderd_public=no

# to setup the DNS TXT records (can be set by bm-tester.conf)
nodes_domain=''
dns_txt='1 2'

# end of configuration

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


# if a config override is in the current directory
cfg=bm-tester.conf
if [ -f "${cfg}" ]
then
  printf 'using configuration override: %s\n' "${cfg}"
  sleep 2
  . "${cfg}"
fi

# possible to re-override nodes-domain from command-line
if [ -n "${1}" ]
then
  old_nd="${nodes_domain}"
  nodes_domain="${1}"
  if [ -n "${old_nd}" -a X"${old_nd}" != "${nodes_domain}" ]
  then
    printf 'command-line override: %s (was: %s}\n' "${nodes_domain}" "${old_nd}"
    sleep 2
  fi
fi

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
CHECK_PROGRAM drill:host
CHECK_PROGRAM awk jq lua52:lua5.2:lua53:lua5.3:lua
CHECK_PROGRAM genbtcltc restart-all-bitmarkds bm-tester
CHECK_PROGRAM generate-bitmarkd-configuration
CHECK_PROGRAM run-bitcoin run-litecoin run-discovery
CHECK_PROGRAM run-bitmarkd run-recorderd
CHECK_PROGRAM make-blockchain node-info

# fail if something is missing
[ X"${ok}" = X"no" ] && ERROR 'missing programs'

# detect GNU getopt
getopt=
case "$(uname)" in
  (FreeBSD|DragonFly)
    getopt=/usr/local/bin/getopt
    ;;
  (NetBSD)
    getopt=/usr/pkg/bin/getopt
    ;;
  (OpenBSD)
    getopt=/usr/local/bin/gnugetopt
    ;;
  (Darwin)
    getopt=/usr/local/opt/gnu-getopt/bin/getopt
    ;;
  (Linux)
    getopt=/usr/bin/getopt
    ;;
  (*)
    ERROR 'OS: %s is not supported' "$(uname)"
    ;;
esac
[ -x "${getopt}" ] || ERROR 'getopt: "%s" is not executable or not installed' "${getopt}"

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
  eval "more_${i}"="\$(( more_${i} + 1 ))"
done

opts=
OPT() {
  opts="${opts} $*"
}

SEP 'expect errors if here:'

CONFIGURE() {
  for i in ${all}
  do
    eval console=\"\${console_${i}}\"
    eval more=\"\${more_${i}:-0}\"
    opts=''
    OPT --chain=local
    OPT --payment=discovery
    OPT "$@"
    OPT --update
    [ X"${recorderd_public}" = X"yes" ] && OPT --recorderd-public
    [ X"${console}" = X"yes" ] && OPT --console
    while [ ${more} -gt 0 ]
    do
      OPT --more
      more=$(( more - 1 ))
    done

    generate-bitmarkd-configuration ${opts} "${i}"
    SEP
  done
}

# first pass configure
CONFIGURE

# print out the dns items
SEP 'configure your local DNS TXT records with the following data\n'
for i in ${dns_txt}
do
  run-bitmarkd --config="%${i}" dns-txt
  SEP
done

# check the TXT records work
SEP 'checking the TXT records...'
for p in drill host
do
  drill=$(which "${p}")
  [ -x "${drill}" ] && break
done
[ -x "${drill}" ] || ERROR 'cannot locate host or drill programs'

r=$(${drill} -t TXT "${nodes_domain}" | grep '^'"${nodes_domain}")
[ -z "${r}" ] && ERROR 'dnsmasq/unbound not setup: missing TXT for: %s' "${nodes_domain}"
printf 'DNS query shows:\n\n'
printf '%s\n\n' "${r}"

# add proper nodes and reconfigure
SEP 'update configuration...'
CONFIGURE --nodes="${nodes_domain}" > /dev/null 2>&1
SEP 'finished'
