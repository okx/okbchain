package mpt

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/okx/brczero/cmd/brczerod/base"

	"log"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

	ethcmn "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/spf13/cobra"

	"github.com/okx/brczero/libs/cosmos-sdk/server"
	"github.com/okx/brczero/libs/cosmos-sdk/store/mpt"
)

func mptViewerCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "viewer [tree] [height]",
		Args:  cobra.ExactArgs(2),
		Short: "iterate mpt store (acc, evm)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkValidKey(args[0])
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("--------- iterate %s data start ---------\n", args[0])
			height, err := strconv.Atoi(args[1])
			if err != nil {
				log.Printf("height error:%s\n", err)
				return
			}
			if height < 0 {
				log.Printf("height can not be negative\n")
				return
			}

			switch args[0] {
			case accStoreKey:
				iterateAccMpt(uint64(height))
			case evmStoreKey:
				iterateEvmMpt(uint64(height))
			}
			log.Printf("--------- iterate %s data end ---------\n", args[0])
		},
	}
	return cmd
}

func iterateAccMpt(height uint64) {
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
	fmt.Println("accTrie root hash:", accTrie.Hash())

	itr := trie.NewIterator(accTrie.NodeIterator(nil))
	for itr.Next() {
		acc := base.DecodeAccount(ethcmn.Bytes2Hex(itr.Key), itr.Value)
		if acc != nil {
			fmt.Printf("%s: %s\n", ethcmn.Bytes2Hex(itr.Key), acc.String())
		}
	}
}

func iterateEvmMpt(height uint64) {
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
	fmt.Println("accTrie root hash:", accTrie.Hash())

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
		fmt.Println(addr.String(), contractTrie.Hash())

		cItr := trie.NewIterator(contractTrie.NodeIterator(nil))
		for cItr.Next() {
			fmt.Printf("%s: %s\n", ethcmn.Bytes2Hex(cItr.Key), ethcmn.Bytes2Hex(cItr.Value))
		}
	}
}
