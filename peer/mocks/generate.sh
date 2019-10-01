#!/bin/sh

list='
github.com/bitmark-inc/bitmarkd/peer/upstream:Upstream:mock_upstream.go
github.com/bitmark-inc/bitmarkd/zmqutil:Client:mock_zmqutil_client.go
'

ERROR() {
  printf 'error: '
  printf "$@"
  printf '\n'
  exit 1
}

copyright=$(mktemp)
cleanup() {
  rm -f "${copyright}"
}
trap cleanup INT EXIT


cat <<EOF > "${copyright}"
SPDX-License-Identifier: ISC
Copyright (c) 2014-2019 Bitmark Inc.
Use of this source code is governed by an ISC
license that can be found in the LICENSE file.
EOF


# directory of this script
out=$(dirname "$0")

# get to project root
while :
do
  [ -f go.mod ] && break
  cd .. || ERROR 'cannot cd to project root'
  [ X"${PWD}" = X"/" ] && ERROR 'cannot cd to project root'
done

# make the mocks
for item in ${list}
do
  src="${item%%:*}"
  item="${item#*:}"
  names="${item%%:*}"
  item="${item#*:}"
  dst="${out}/${item%%:*}"
  item="${item#*:}"

  printf '%s (%s) → %s\n' "${src}" "${names}" "${dst}"

  mockgen -copyright_file="${copyright}" -package=mocks "${src}" ${names} > "${dst}"

done
