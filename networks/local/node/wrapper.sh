#!/usr/bin/env sh

##
## Input parameters
##
ID=${ID:-0}
LOG=${LOG:-okbchaind.log}

##
## Run binary with all parameters
##
export EXCHAINDHOME="/okbchaind/node${ID}/okbchaind"

if [ -d "$(dirname "${EXCHAINDHOME}"/"${LOG}")" ]; then
  okbchaind --chain-id okbchain-1 --home "${EXCHAINDHOME}" "$@" | tee "${EXCHAINDHOME}/${LOG}"
else
  okbchaind --chain-id okbchain-1 --home "${EXCHAINDHOME}" "$@"
fi

