#!/bin/sh
# generate all LOCAL bitmarkd configuration configurations

all='1 2 3 4 5 6 7 8 9'
console='1 2 8'
more='1 2 8'

# to setup the DNS TXT records
dns_txt='1 2'
nodes_domain='nodes.test.bitmark.com'


ERROR() {
  printf 'error: '
  printf "$@"
  printf '\n'
  exit 1
}


# main program

[ -n "${1}" ] && nodes_domain="${1}"

xdg_home="${XDG_CONFIG_HOME}"
[ -z "${xdg_home}" ] && xdg_home="${HOME}/.config"
[ -d "${xdg_home}" ] || ERROR 'missing directory: "%s" please create first' "${xdg_home}"

this_dir=$(dirname "$0")
PATH="${this_dir}:${PATH}"
samples="${this_dir}/samples"

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

subs=
for i in ${all}
do
  eval console=\"\${console_${i}}\"
  eval more=\"\${more_${i}}\"
  opts=''
  OPT --chain=local
  OPT --nodes="${nodes_domain}"
  OPT --bitcoin="${xdg_home}/bitcoin"
  OPT --litecoin="${xdg_home}/litecoin"
  OPT --discovery="${xdg_home}/discovery"
  OPT --update
  [ X"${console}" = X"yes" ] && OPT --console
  [ X"${more}" = X"yes" ] && OPT --more

  generate-bitmarkd-configuration ${opts} "${i}"

  public=$(cat "${xdg_home}/bitmarkd${i}/proof.public")
  subs="${subs}s/%%BITMARKD_${i}%%/${public#PUBLIC:}/;"
done

# fixup recorderd
for program in recorderd
do
  d="${xdg_home}/${program}"
  dcf="${d}/${program}.conf"
  [ -f "${dcf}" ] || ERROR 'missing file: %s' "${dcf}"
  sed -E -i .bk "${subs}" "${dcf}"
done

# print out the dns items
if [ X"${nodes_domain}" = X"nodes.test.bitmark.com" ]
then
  printf '==================================================\n'
  printf 'configure you local DNS TXT records with the following data\n'
  printf 'then re-run this configuration with the node domain\n\n'
  printf '    e.g.   %s nodes.localdomain\n\n' "${0}"

  for i in ${dns_txt}
  do
    run-bitmarkd --config="%${i}" dns-txt
    printf '==================================================\n'
  done
fi
