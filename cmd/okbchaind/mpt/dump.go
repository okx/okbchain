package mpt

import (
	"encoding/binary"
	"fmt"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/okx/okbchain/cmd/okbchaind/base"
	"github.com/okx/okbchain/libs/cosmos-sdk/server"
	"github.com/okx/okbchain/libs/cosmos-sdk/store/mpt"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

func dumpStorageCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump-storage [height]",
		Args:  cobra.ExactArgs(1),
		Short: "dump mpt storage",
		Run: func(cmd *cobra.Command, args []string) {
			height, err := strconv.Atoi(args[0])
			if err != nil {
				log.Printf("height error:%s\n", err)
				return
			}
			if height < 0 {
				log.Printf("height can not be negative\n")
				return
			}
			dumpStorage(uint64(height))
		},
	}
	return cmd
}

func dumpStorage(height uint64) {
	accMptDb := mpt.InstanceOfMptStore()
	heightBytes, err := accMptDb.TrieDB().DiskDB().Get(mpt.KeyPrefixAccLatestStoredHeight)
	panicError(err)
	lastestHeight := binary.BigEndian.Uint64(heightBytes)
	if lastestHeight < height {
		panic(fmt.Errorf("height(%d) > lastestHeight(%d)", height, lastestHeight))
	}
	if height == 0 {
		height = lastestHeight
	}

	hhash := sdk.Uint64ToBigEndian(height)
	rootHash, err := accMptDb.TrieDB().DiskDB().Get(append(mpt.KeyPrefixAccRootMptHash, hhash...))
	panicError(err)
	accTrie, err := accMptDb.OpenTrie(ethcmn.BytesToHash(rootHash))
	panicError(err)

	var stateRoot ethcmn.Hash
	itr := trie.NewIterator(accTrie.NodeIterator(nil))
	for itr.Next() {
		addr := ethcmn.BytesToAddress(accTrie.GetKey(itr.Key))
		addrHash := ethcrypto.Keccak256Hash(addr[:])
		acc := base.DecodeAccount(addr.String(), itr.Value)
		if acc == nil {
			continue
		}
		stateRoot.SetBytes(acc.GetStateRoot().Bytes())

		contractTrie := getStorageTrie(accMptDb, addrHash, stateRoot)

		cItr := trie.NewIterator(contractTrie.NodeIterator(nil))
		for cItr.Next() {
			fmt.Printf("%s_%s,%s\n", addr, ethcmn.Bytes2Hex(cItr.Key), ethcmn.Bytes2Hex(cItr.Value))
		}
	}
}
