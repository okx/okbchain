package keeper

import (
	"math/big"

	"github.com/okx/okbchain/x/evm/watcher"

	tmtypes "github.com/okx/okbchain/libs/tendermint/types"

	"github.com/ethereum/go-ethereum/common"

	abci "github.com/okx/okbchain/libs/tendermint/abci/types"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/okx/okbchain/x/evm/types"
)

// BeginBlock sets the block hash -> block height map for the previous block height
// and resets the Bloom filter and the transaction count to 0.
func (k *Keeper) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {

	if req.Header.GetHeight() < 1 {
		return
	}

	// Gas costs are handled within msg handler so costs should be ignored
	ctx.SetGasMeter(sdk.NewInfiniteGasMeter())

	// Set the hash -> height and height -> hash mapping.
	currentHash := req.Hash
	height := req.Header.GetHeight()
	blockHash := common.BytesToHash(currentHash)
	k.SetHeightHash(ctx, uint64(height), blockHash)
	k.SetBlockHeight(ctx, currentHash, height)
	// Add latest block height and hash to cache
	k.AddHeightHashToCache(height, blockHash.Hex())
	// Add latest block height and hash to cache
	// reset counters that are used on CommitStateDB.Prepare
	if !ctx.IsTraceTx() {
		k.Bloom = big.NewInt(0)
		k.TxCount = 0
		k.LogSize = 0
		k.LogsManages.Reset()
		k.Bhash = blockHash

		//that can make sure latest block has been committed
		k.UpdatedAccount = k.UpdatedAccount[:0]
		k.Watcher.NewHeight(uint64(req.Header.GetHeight()), blockHash, req.Header)
	}

	if tmtypes.DownloadDelta {
		types.GetEvmParamsCache().SetNeedParamsUpdate()
		types.GetEvmParamsCache().SetNeedBlockedUpdate()
	}
}

// EndBlock updates the accounts and commits state objects to the KV Store, while
// deleting the empty ones. It also sets the bloom filers for the request block to
// the store. The EVM end block logic doesn't update the validator set, thus it returns
// an empty slice.
func (k *Keeper) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	// Gas costs are handled within msg handler so costs should be ignored
	ctx.SetGasMeter(sdk.NewInfiniteGasMeter())

	// set the block bloom filter bytes to store
	bloom := ethtypes.BytesToBloom(k.Bloom.Bytes())

	k.SetBlockBloom(ctx, req.Height, bloom)
	if types.GetEnableBloomFilter() {
		// the hash of current block is stored when executing BeginBlock of next block.
		// so update section in the next block.
		if indexer := types.GetIndexer(); indexer != nil {
			if types.GetIndexer().IsProcessing() {
				// notify new height
				go func() {
					indexer.NotifyNewHeight(ctx)
				}()
			} else {
				interval := uint64(req.Height - tmtypes.GetStartBlockHeight())
				if interval >= (indexer.GetValidSections()+1)*types.BloomBitsBlocks {
					go types.GetIndexer().ProcessSection(ctx, k, interval, k.Watcher.GetBloomDataPoint())
				}
			}
		}
	}

	if watcher.IsWatcherEnabled() && k.Watcher.IsFirstUse() {
		store := k.GetParamSubspace().CustomKVStore(ctx)
		iteratorBlockedList := sdk.KVStorePrefixIterator(store, types.KeyPrefixContractBlockedList)
		defer iteratorBlockedList.Close()
		for ; iteratorBlockedList.Valid(); iteratorBlockedList.Next() {
			vaule := iteratorBlockedList.Value()
			if len(vaule) == 0 {
				k.Watcher.SaveContractBlockedListItem(iteratorBlockedList.Key()[1:])
			} else {
				k.Watcher.SaveContractMethodBlockedListItem(iteratorBlockedList.Key()[1:], vaule)
			}
		}

		iteratorDeploymentWhitelist := sdk.KVStorePrefixIterator(store, types.KeyPrefixContractDeploymentWhitelist)
		defer iteratorDeploymentWhitelist.Close()
		for ; iteratorDeploymentWhitelist.Valid(); iteratorDeploymentWhitelist.Next() {
			k.Watcher.SaveContractDeploymentWhitelistItem(iteratorDeploymentWhitelist.Key()[1:])
		}

		k.Watcher.Used()
	}

	if watcher.IsRpcNode() {
		// eth block
		var evmTxs []common.Hash
		var gasUsed int64
		for _, txRes := range req.DeliverTxs {
			if txRes.GetType() == int(sdk.EvmTxType) {
				gasUsed += txRes.GasUsed
				evmTxs = append(evmTxs, common.BytesToHash(txRes.GetHash()))
			}
		}
		block, ethBlockHash := watcher.NewBlock(uint64(req.Height), bloom,
			ctx.BlockHeader(), uint64(0xffffffff), big.NewInt(gasUsed), evmTxs)

		k.SetEthBlockByHeight(ctx, uint64(req.Height), block)
		k.SetEthBlockByHash(ctx, ethBlockHash.Bytes(), block)

		if watcher.IsWatcherEnabled() {
			params := k.GetParams(ctx)
			k.Watcher.SaveParams(params)

			k.Watcher.SaveBlock(block)
			k.Watcher.SaveBlockStdTxHash()
		}
	}

	k.UpdateInnerBlockData()

	return []abci.ValidatorUpdate{}
}
