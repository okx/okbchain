#!/bin/bash
nohup okbchaind start \
	--rpc.unsafe \
	--local-rpc-port 26657 \
	--log_level "main:info,*:error" \
	--log_file json \
	--dynamic-gp-mode=2 \
	--consensus.timeout_commit 3.5s \
	--enable-preruntx=1 \
	--enable-gid \
	--fast-query=false \
	--append-pid=true \
	--iavl-output-modules evm=0,acc=0 \
	--trie.dirty-disabled=true \
	--trace --home ./_cache_evm \
	--chain-id okbchain-67 \
	--elapsed Round=1,CommitRound=1,Produce=1 \
	--deliver-txs-mode=2 \
	--mempool.max_tx_num_per_block=20000 \
	--mempool.size=200000 \
	--tree-enable-async-commit=false \
	--commit-gap-height 3 \
	&
