package rootmulti

import (
	"github.com/okx/brczero/libs/cosmos-sdk/store/dbadapter"
	"github.com/okx/brczero/libs/cosmos-sdk/store/types"
	"github.com/okx/brczero/libs/iavl"
)

var commithash = []byte("FAKE_HASH")

//----------------------------------------
// commitDBStoreWrapper should only be used for simulation/debugging,
// as it doesn't compute any commit hash, and it cannot load older state.

// Wrapper type for dbm.Db with implementation of KVStore
type commitDBStoreAdapter struct {
	dbadapter.Store
}

func (cdsa commitDBStoreAdapter) Commit(*iavl.TreeDelta, []byte) (types.CommitID, iavl.TreeDelta, []byte) {
	return types.CommitID{
		Version: -1,
		Hash:    commithash,
	}, iavl.TreeDelta{}, nil
}

func (cdsa commitDBStoreAdapter) CommitterCommit(interface{}) (types.CommitID, interface{}) {
	return types.CommitID{
		Version: -1,
		Hash:    commithash,
	}, &iavl.TreeDelta{}
}

func (cdsa commitDBStoreAdapter) LastCommitID() types.CommitID {
	return types.CommitID{
		Version: -1,
		Hash:    commithash,
	}
}

func (cdsa commitDBStoreAdapter) LastCommitVersion() (v int64) {
	return -1
}

func (cdsa commitDBStoreAdapter) SetPruning(_ types.PruningOptions) {}

func (cdsa commitDBStoreAdapter) GetDBReadTime() int   { return 0 }
func (cdsa commitDBStoreAdapter) GetDBWriteCount() int { return 0 }

func (cdsa commitDBStoreAdapter) GetDBReadCount() int   { return 0 }
func (cdsa commitDBStoreAdapter) GetNodeReadCount() int { return 0 }

func (cdsa commitDBStoreAdapter) ResetCount() {}

func (cdsa commitDBStoreAdapter) GetFlatKVReadTime() int   { return 0 }
func (cdsa commitDBStoreAdapter) GetFlatKVWriteTime() int  { return 0 }
func (cdsa commitDBStoreAdapter) GetFlatKVReadCount() int  { return 0 }
func (cdsa commitDBStoreAdapter) GetFlatKVWriteCount() int { return 0 }
func (cdsa commitDBStoreAdapter) SetUpgradeVersion(int64)  {}
