#!/bin/sh
# this sets a default easy to guess password


ERROR() {
  printf 'error: '
  printf "$@"
  printf '\n'
  exit 1
}


# main program

password=1234567890
[ -n "${1}" ] && password="${1}"

xdg_home="${XDG_CONFIG_HOME}"
[ -z "${xdg_home}" ] && xdg_home="${HOME}/.config"
[ -d "${xdg_home}" ] || ERROR 'missing directory: "%s" please create first' "${xdg_home}"

# create first and second users
bitmark-cli --network=local --identity=first --password="${password}" setup --connect=127.0.0.1:22130 --description='first user' --new
bitmark-cli --network=local --identity=second --password="${password}" add --description='second user' --new

# create users for all bitmarkds
for i in 1 2 3 4 5 6 7 8 9
do
  id="node-${i}"
  seed="${xdg_home}/bitmarkd${i}/proof.test"
  [ -f "${seed}" ] || ERROR 'missing file: %s' "${seed}"
  bitmark-cli --network=local --identity="${id}" --password="${password}" add --description="node ${i}" --seed="$(cat "${seed}")"
done
