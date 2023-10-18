#!/usr/bin/env bash

set -e

./testnet.sh -s -i -n 5 -r 1

okbchaincli keys add --recover val0 -m "puzzle glide follow cruel say burst deliver wild tragic galaxy lumber offer" --coin-type 996 -y
okbchaincli keys add --recover val1 -m "palace cube bitter light woman side pave cereal donor bronze twice work" --coin-type 996 -y
okbchaincli keys add --recover val2 -m "antique onion adult slot sad dizzy sure among cement demise submit scare" --coin-type 996 -y
okbchaincli keys add --recover val3 -m "lazy cause kite fence gravity regret visa fuel tone clerk motor rent" --coin-type 996 -y

sleep 4

okbchaincli tx gov submit-proposal upgrade ../proposals/wasm.proposal --from val0 --fees 1okb  -y -b block
okbchaincli tx gov vote 1 yes --from val0 --fees 0.01okb  -y -b block
okbchaincli tx gov vote 1 yes --from val1 --fees 0.01okb  -y -b block
okbchaincli tx gov vote 1 yes --from val2 --fees 0.01okb  -y -b block
okbchaincli tx gov vote 1 yes --from val3 --fees 0.01okb  -y -b block

#sleep 5
#
#./addnewnode.sh -n 4
