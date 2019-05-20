#!/bin/sh

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
  btc {
    type = "daemon"
    node = "localhost:18001"
    user = "btcuser1"
    pass = "beis7uvei9ALei4ofeu6ahFaeQu0IephTheebuchuuXio5ia"
  }

  ltc {
    type = "daemon"
    node = "localhost:19001"
    user = "litecoinuser"
    pass = "gdbhrkztqxgnfsggqzpzsxrmkgzvksfjwgngwgjsgqjknrnqspxgrvrmxdwxbbmt"
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
      run=run-bitcoin
      ;;
    (ltc)
      run=run-litecoin
      ;;
    (*)
      ERROR 'unknown currency: %s' "${c}"
      ;;
  esac

  printf '==================================================\n'
  bitmark-wallet --conf "${conf}" "${c}" sync --testnet
  bitmark-wallet --conf "${conf}" "${c}" newaddress --testnet | (
    while read tag address junk
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
