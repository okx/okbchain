#!/bin/bash

OPTIONS="--from captain --gas-prices 0.0000000001okt --gas auto -b block --gas-adjustment 1.5 -y"

okbchaincli tx wasm store ./counter.wasm ${OPTIONS}
okbchaincli tx wasm instantiate 1 '{}' ${OPTIONS}

# ex14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s6fqu27
# 0xbbE4733d85bc2b90682147779DA49caB38C0aA1F
okbchaincli tx wasm execute 0x5A8D648DEE57b2fc90D98DC17fa887159b69638b '{"add":{"delta":"16"}}'  ${OPTIONS}

okbchaincli query wasm contract-state smart 0x5A8D648DEE57b2fc90D98DC17fa887159b69638b '{"get_counter":{}}'

