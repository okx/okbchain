package mempool

import (
	"encoding/binary"
	"path/filepath"
	"sync"

	"github.com/okx/brczero/libs/cosmos-sdk/client/flags"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	cfg "github.com/okx/brczero/libs/tendermint/config"
	db "github.com/okx/brczero/libs/tm-db"
	"github.com/spf13/viper"
)

const (
	pguDBDir  = "data"
	pguDBName = "pgu"
)

var (
	pguDB   db.DB
	pguOnce sync.Once
)

func initDB() {
	homeDir := viper.GetString(flags.FlagHome)
	dbPath := filepath.Join(homeDir, pguDBDir)
	var err error
	pguDB, err = sdk.NewDB(pguDBName, dbPath)
	if err != nil {
		panic(err)
	}
}

func updatePGU(txHash []byte, gas int64) error {
	if !cfg.DynamicConfig.GetPGUPersist() {
		return nil
	}
	pguOnce.Do(initDB)
	bytesGas := make([]byte, 8)
	binary.BigEndian.PutUint64(bytesGas, uint64(gas))
	return pguDB.Set(txHash, bytesGas)
}

func getPGUGas(txHash []byte) int64 {
	pguOnce.Do(initDB)
	data, err := pguDB.Get(txHash)
	if err != nil || len(data) == 0 {
		return -1
	}
	return int64(binary.BigEndian.Uint64(data))
}
