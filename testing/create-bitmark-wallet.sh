#!/bin/sh

ERROR() {
  printf 'error: '
  # shellcheck disable=SC2059
  printf -- "$@"
  printf '\n'
  exit 1
}


# main program

xdg_home="${XDG_CONFIG_HOME}"
[ -z "${xdg_home}" ] && xdg_home="${HOME}/.config"
[ -d "${xdg_home}" ] || ERROR 'missing directory: "%s" please create first' "${xdg_home}"


d="${xdg_home}/bitmark-wallet/local"
mkdir -p "${d}"
conf="${d}/local-bitmark-wallet.conf"

export WALLET_PASSWORD='1234567890'

if [ ! -f "${conf}" ]
then
   cat > "${conf}" <<EOF
# bitmark-wallet.conf

datadir = "."
walletdb = "wallet.dat"

agent {
  btc {agent {
  btc {
    type = "daemon"
    node = "localhost:18443"
    user = "asgymsscumjvwluanhpmzhhc"
    pass = "cuh-OAHtwo6-4ZsXvm8syxthCmV8oyHY07T1oHm18-c="
  }

  ltc {
    type = "daemon"
    node = "localhost:19443"
    user = "xehjpioqyawynunxknoawpun"
    pass = "OGjFWI5zkuVTQ4fdiCfbPvRO8hJN2gV_qv0syEDr-6g="
  }
}

EOF

   bitmark-wallet --conf "${conf}" init
fi

currencies='btc ltc'

# sync each currency
for c in ${currencies}
do
  case "${c}" in
    (btc)
      run='run-bitcoin'
      ;;
    (ltc)
      run='run-litecoin'
      ;;
    (*)
      ERROR 'unknown currency: %s' "${c}"
      ;;
  esac

  printf '==================================================\n'
  bitmark-wallet --conf "${conf}" "${c}" sync --testnet
  bitmark-wallet --conf "${conf}" "${c}" newaddress --testnet | (
    # shellcheck disable=SC2034
    while read -r tag address junk
    do
      [ X"${tag}" = X"Address:" ] && ${run} sendtoaddress "${address}" 25
    done
  )
done

genbtcltc

for c in ${currencies}
do
  printf '==================================================\n'
  bitmark-wallet --conf "${conf}" "${c}" sync --testnet
done
