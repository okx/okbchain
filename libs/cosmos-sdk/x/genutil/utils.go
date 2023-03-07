package genutil

import (
	"encoding/json"
	"path/filepath"
	"time"

	cfg "github.com/okx/okbchain/libs/tendermint/config"
	"github.com/okx/okbchain/libs/tendermint/crypto"
	tmos "github.com/okx/okbchain/libs/tendermint/libs/os"
	"github.com/okx/okbchain/libs/tendermint/p2p"
	"github.com/okx/okbchain/libs/tendermint/privval"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
)

// ExportGenesisFile creates and writes the genesis configuration to disk. An
// error is returned if building or writing the configuration to file fails.
func ExportGenesisFile(genDoc *tmtypes.GenesisDoc, genFile string) error {
	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genFile)
}

// ExportGenesisFileWithTime creates and writes the genesis configuration to disk.
// An error is returned if building or writing the configuration to file fails.
func ExportGenesisFileWithTime(
	genFile, chainID string, validators []tmtypes.GenesisValidator,
	appState json.RawMessage, genTime time.Time,
) error {

	genDoc := tmtypes.GenesisDoc{
		GenesisTime: genTime,
		ChainID:     chainID,
		Validators:  validators,
		AppState:    appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genFile)
}

func InitializeNodeValidatorFilesByIndex(config *cfg.Config, index int) (nodeID string, valPubKey crypto.PubKey, err error) {
	nodeKey, err := p2p.LoadOrGenNodeKeyByIndex(config.NodeKeyFile(), index)
	if err != nil {
		return "", nil, err
	}

	return initValidatorFiles(config, nodeKey)
}

// InitializeNodeValidatorFiles creates private validator and p2p configuration files.
func InitializeNodeValidatorFiles(config *cfg.Config) (nodeID string, valPubKey crypto.PubKey, err error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return "", nil, err
	}

	return initValidatorFiles(config, nodeKey)
}

func initValidatorFiles(config *cfg.Config, nodeKey *p2p.NodeKey) (nodeID string, valPubKey crypto.PubKey, err error) {

	nodeID = string(nodeKey.ID())

	pvKeyFile := config.PrivValidatorKeyFile()
	if err := tmos.EnsureDir(filepath.Dir(pvKeyFile), 0777); err != nil {
		return "", nil, err
	}

	pvStateFile := config.PrivValidatorStateFile()
	if err := tmos.EnsureDir(filepath.Dir(pvStateFile), 0777); err != nil {
		return "", nil, err
	}

	valPubKey, err = privval.LoadOrGenFilePV(pvKeyFile, pvStateFile).GetPubKey()
	if err != nil {
		return "", nil, err
	}

	return nodeID, valPubKey, nil
}
