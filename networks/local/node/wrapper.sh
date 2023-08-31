#!/usr/bin/env sh

##
## Input parameters
##
ID=${ID:-0}
LOG=${LOG:-brczerod.log}

##
## Run binary with all parameters
##
export OKBCHAINDHOME="/okbchaind/node${ID}/okbchaind"

if [ -d "$(dirname "${OKBCHAINDHOME}"/"${LOG}")" ]; then
  brczerod --chain-id okbchain-1 --home "${OKBCHAINDHOME}" "$@" | tee "${OKBCHAINDHOME}/${LOG}"
else
  brczerod --chain-id okbchain-1 --home "${OKBCHAINDHOME}" "$@"
fi

