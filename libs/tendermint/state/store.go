package state

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tendermint/go-amino"

	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	tmmath "github.com/okx/brczero/libs/tendermint/libs/math"
	tmos "github.com/okx/brczero/libs/tendermint/libs/os"
	"github.com/okx/brczero/libs/tendermint/types"
	dbm "github.com/okx/brczero/libs/tm-db"
	"github.com/pkg/errors"
)

const (
	// persist validators every valSetCheckpointInterval blocks to avoid
	// LoadValidators taking too much time.
	// https://github.com/tendermint/tendermint/pull/3438
	// 100000 results in ~ 100ms to get 100 validators (see BenchmarkLoadValidators)
	valSetCheckpointInterval = 100000
)

//------------------------------------------------------------------------

func calcValidatorsKey(height int64) []byte {
	return []byte(fmt.Sprintf("validatorsKey:%v", height))
}

func calcConsensusParamsKey(height int64) []byte {
	return []byte(fmt.Sprintf("consensusParamsKey:%v", height))
}

func calcABCIResponsesKey(height int64) []byte {
	return []byte(fmt.Sprintf("abciResponsesKey:%v", height))
}

// LoadStateFromDBOrGenesisFile loads the most recent state from the database,
// or creates a new one from the given genesisFilePath and persists the result
// to the database.
func LoadStateFromDBOrGenesisFile(stateDB dbm.DB, genesisFilePath string) (State, error) {
	state := LoadState(stateDB)
	if state.IsEmpty() {
		var err error
		state, err = MakeGenesisStateFromFile(genesisFilePath)
		if err != nil {
			return state, err
		}
		SaveState(stateDB, state)
	}

	return state, nil
}

// LoadStateFromDBOrGenesisDoc loads the most recent state from the database,
// or creates a new one from the given genesisDoc and persists the result
// to the database.
func LoadStateFromDBOrGenesisDoc(stateDB dbm.DB, genesisDoc *types.GenesisDoc) (State, error) {
	state := LoadState(stateDB)
	if state.IsEmpty() {
		var err error
		state, err = MakeGenesisState(genesisDoc)
		if err != nil {
			return state, err
		}
		SaveState(stateDB, state)
	}

	return state, nil
}

// LoadState loads the State from the database.
func LoadState(db dbm.DB) State {
	return loadState(db, stateKey)
}

func loadState(db dbm.DB, key []byte) (state State) {
	buf, err := db.Get(key)
	if err != nil {
		panic(err)
	}
	if len(buf) == 0 {
		return state
	}

	err = cdc.UnmarshalBinaryBare(buf, &state)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		tmos.Exit(fmt.Sprintf(`LoadState: Data has been corrupted or its spec has changed:
                %v\n`, err))
	}
	// TODO: ensure that buf is completely read.

	return state
}

// SaveState persists the State, the ValidatorsInfo, and the ConsensusParamsInfo to the database.
// This flushes the writes (e.g. calls SetSync).
func SaveState(db dbm.DB, state State) {
	saveState(db, state, stateKey)
}

func saveState(db dbm.DB, state State, key []byte) {
	nextHeight := state.LastBlockHeight + 1
	// If first block, save validators for block 1.
	if nextHeight == types.GetStartBlockHeight()+1 {
		// This extra logic due to Tendermint validator set changes being delayed 1 block.
		// It may get overwritten due to InitChain validator updates.
		lastHeightVoteChanged := types.GetStartBlockHeight() + 1
		saveValidatorsInfo(db, nextHeight, lastHeightVoteChanged, state.Validators)
	}
	// Save next validators.
	saveValidatorsInfo(db, nextHeight+1, state.LastHeightValidatorsChanged, state.NextValidators)
	// Save next consensus params.
	saveConsensusParamsInfo(db, nextHeight, state.LastHeightConsensusParamsChanged, state.ConsensusParams)
	db.SetSync(key, state.Bytes())
}

//------------------------------------------------------------------------

// ABCIResponses retains the responses
// of the various ABCI calls during block processing.
// It is persisted to disk for each height before calling Commit.
type ABCIResponses struct {
	DeliverTxs []*abci.ResponseDeliverTx `json:"deliver_txs"`
	EndBlock   *abci.ResponseEndBlock    `json:"end_block"`
	BeginBlock *abci.ResponseBeginBlock  `json:"begin_block"`
}

func (arz ABCIResponses) AminoSize(cdc *amino.Codec) int {
	size := 0
	for _, tx := range arz.DeliverTxs {
		txSize := tx.AminoSize(cdc)
		size += 1 + amino.UvarintSize(uint64(txSize)) + txSize
	}
	if arz.EndBlock != nil {
		endBlockSize := arz.EndBlock.AminoSize(cdc)
		size += 1 + amino.UvarintSize(uint64(endBlockSize)) + endBlockSize
	}
	if arz.BeginBlock != nil {
		beginBlockSize := arz.BeginBlock.AminoSize(cdc)
		size += 1 + amino.UvarintSize(uint64(beginBlockSize)) + beginBlockSize
	}
	return size
}

func (arz ABCIResponses) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(arz.AminoSize(cdc))
	err := arz.MarshalAminoTo(cdc, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (arz ABCIResponses) MarshalAminoTo(cdc *amino.Codec, buf *bytes.Buffer) error {
	var err error
	// field 1
	for i := 0; i < len(arz.DeliverTxs); i++ {
		const pbKey = 1<<3 | 2
		buf.WriteByte(pbKey)
		txSize := arz.DeliverTxs[i].AminoSize(cdc)
		err = amino.EncodeUvarintToBuffer(buf, uint64(txSize))
		if err != nil {
			return err
		}
		lenBeforeData := buf.Len()
		err = arz.DeliverTxs[i].MarshalAminoTo(cdc, buf)
		if err != nil {
			return err
		}
		if buf.Len()-lenBeforeData != txSize {
			return amino.NewSizerError(arz.DeliverTxs[i], buf.Len()-lenBeforeData, txSize)
		}
	}
	// field 2
	if arz.EndBlock != nil {
		const pbKey = 2<<3 | 2
		buf.WriteByte(pbKey)
		endBlockSize := arz.EndBlock.AminoSize(cdc)
		err = amino.EncodeUvarintToBuffer(buf, uint64(endBlockSize))
		if err != nil {
			return err
		}
		lenBeforeData := buf.Len()
		err = arz.EndBlock.MarshalAminoTo(cdc, buf)
		if err != nil {
			return err
		}
		if buf.Len()-lenBeforeData != endBlockSize {
			return amino.NewSizerError(arz.EndBlock, buf.Len()-lenBeforeData, endBlockSize)
		}
	}
	// field 3
	if arz.BeginBlock != nil {
		const pbKey = 3<<3 | 2
		buf.WriteByte(pbKey)
		beginBlockSize := arz.BeginBlock.AminoSize(cdc)
		err = amino.EncodeUvarintToBuffer(buf, uint64(beginBlockSize))
		if err != nil {
			return err
		}
		lenBeforeData := buf.Len()
		err = arz.BeginBlock.MarshalAminoTo(cdc, buf)
		if err != nil {
			return err
		}
		if buf.Len()-lenBeforeData != beginBlockSize {
			return amino.NewSizerError(arz.BeginBlock, buf.Len()-lenBeforeData, beginBlockSize)
		}
	}

	return nil
}

// UnmarshalFromAmino unmarshal data from amino bytes.
func (arz *ABCIResponses) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]
		if len(data) == 0 {
			break
		}
		pos, pbType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if pbType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, _ = amino.DecodeUvarint(data)

			data = data[n:]
			if len(data) < int(dataLen) {
				return errors.New("not enough data")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			var resDeliverTx *abci.ResponseDeliverTx = nil
			if len(subData) != 0 {
				resDeliverTx = &abci.ResponseDeliverTx{}
				err := resDeliverTx.UnmarshalFromAmino(cdc, subData)
				if err != nil {
					return err
				}
			}
			arz.DeliverTxs = append(arz.DeliverTxs, resDeliverTx)

		case 2:
			eBlock := &abci.ResponseEndBlock{}
			if len(subData) != 0 {
				err := eBlock.UnmarshalFromAmino(cdc, subData)
				if err != nil {
					return err
				}
			}
			arz.EndBlock = eBlock

		case 3:
			bBlock := &abci.ResponseBeginBlock{}
			if len(subData) != 0 {
				err := bBlock.UnmarshalFromAmino(cdc, subData)
				if err != nil {
					return err
				}
			}
			arz.BeginBlock = bBlock

		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

// PruneStates deletes states between the given heights (including from, excluding to). It is not
// guaranteed to delete all states, since the last checkpointed state and states being pointed to by
// e.g. `LastHeightChanged` must remain. The state at to must also exist.
//
// The from parameter is necessary since we can't do a key scan in a performant way due to the key
// encoding not preserving ordering: https://github.com/tendermint/tendermint/issues/4567
// This will cause some old states to be left behind when doing incremental partial prunes,
// specifically older checkpoints and LastHeightChanged targets.
func PruneStates(db dbm.DB, from int64, to int64) error {
	if from <= 0 || to <= 0 {
		return fmt.Errorf("from height %v and to height %v must be greater than 0", from, to)
	}
	if from >= to {
		return fmt.Errorf("from height %v must be lower than to height %v", from, to)
	}
	valInfo := loadValidatorsInfo(db, to)
	if valInfo == nil {
		return fmt.Errorf("validators at height %v not found", to)
	}
	paramsInfo := loadConsensusParamsInfo(db, to)
	if paramsInfo == nil {
		return fmt.Errorf("consensus params at height %v not found", to)
	}

	keepVals := make(map[int64]bool)
	if valInfo.ValidatorSet == nil {
		keepVals[valInfo.LastHeightChanged] = true
		keepVals[lastStoredHeightFor(to, valInfo.LastHeightChanged)] = true // keep last checkpoint too
	}
	keepParams := make(map[int64]bool)
	if paramsInfo.ConsensusParams.Equals(&types.ConsensusParams{}) {
		keepParams[paramsInfo.LastHeightChanged] = true
	}

	batch := db.NewBatch()
	defer batch.Close()
	pruned := uint64(0)
	var err error

	// We have to delete in reverse order, to avoid deleting previous heights that have validator
	// sets and consensus params that we may need to retrieve.
	for h := to - 1; h >= from; h-- {
		// For heights we keep, we must make sure they have the full validator set or consensus
		// params, otherwise they will panic if they're retrieved directly (instead of
		// indirectly via a LastHeightChanged pointer).
		if keepVals[h] {
			v := loadValidatorsInfo(db, h)
			if v.ValidatorSet == nil {
				v.ValidatorSet, err = LoadValidators(db, h)
				if err != nil {
					return err
				}
				v.LastHeightChanged = h
				batch.Set(calcValidatorsKey(h), v.Bytes())
			}
		} else {
			batch.Delete(calcValidatorsKey(h))
		}

		if keepParams[h] {
			p := loadConsensusParamsInfo(db, h)
			if p.ConsensusParams.Equals(&types.ConsensusParams{}) {
				p.ConsensusParams, err = LoadConsensusParams(db, h)
				if err != nil {
					return err
				}
				p.LastHeightChanged = h
				batch.Set(calcConsensusParamsKey(h), p.Bytes())
			}
		} else {
			batch.Delete(calcConsensusParamsKey(h))
		}

		batch.Delete(calcABCIResponsesKey(h))
		pruned++

		// avoid batches growing too large by flushing to database regularly
		if pruned%1000 == 0 && pruned > 0 {
			err := batch.Write()
			if err != nil {
				return err
			}
			batch.Close()
			batch = db.NewBatch()
			defer batch.Close()
		}
	}

	err = batch.WriteSync()
	if err != nil {
		return err
	}

	return nil
}

// NewABCIResponses returns a new ABCIResponses
func NewABCIResponses(block *types.Block) *ABCIResponses {
	resDeliverTxs := make([]*abci.ResponseDeliverTx, len(block.Data.Txs))
	if len(block.Data.Txs) == 0 {
		// This makes Amino encoding/decoding consistent.
		resDeliverTxs = nil
	}
	return &ABCIResponses{
		DeliverTxs: resDeliverTxs,
	}
}

// Bytes serializes the ABCIResponse using go-amino.
func (arz *ABCIResponses) Bytes() []byte {
	bz, err := arz.MarshalToAmino(cdc)
	if err != nil {
		return cdc.MustMarshalBinaryBare(arz)
	}
	return bz
}

func (arz *ABCIResponses) ResultsHash() []byte {
	results := types.NewResults(arz.DeliverTxs)
	return results.Hash()
}
func (arz *ABCIResponses) String() string {
	str := strings.Builder{}
	results := types.NewResults(arz.DeliverTxs)
	for _, v := range results {
		str.WriteString(fmt.Sprintf("code:%d,msg:=%s\n", v.Code, v.Data.String()))
	}
	return str.String()
}

// LoadABCIResponses loads the ABCIResponses for the given height from the database.
// This is useful for recovering from crashes where we called app.Commit and before we called
// s.Save(). It can also be used to produce Merkle proofs of the result of txs.
func LoadABCIResponses(db dbm.DB, height int64) (*ABCIResponses, error) {
	buf, err := db.Get(calcABCIResponsesKey(height))
	if err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return nil, ErrNoABCIResponsesForHeight{height}
	}

	abciResponses := new(ABCIResponses)
	err = cdc.UnmarshalBinaryBare(buf, abciResponses)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		tmos.Exit(fmt.Sprintf(`LoadABCIResponses: Data has been corrupted or its spec has
                changed: %v\n`, err))
	}
	// TODO: ensure that buf is completely read.

	return abciResponses, nil
}

// SaveABCIResponses persists the ABCIResponses to the database.
// This is useful in case we crash after app.Commit and before s.Save().
// Responses are indexed by height so they can also be loaded later to produce
// Merkle proofs.
//
// Exposed for testing.
func SaveABCIResponses(db dbm.DB, height int64, abciResponses *ABCIResponses) {
	db.SetSync(calcABCIResponsesKey(height), abciResponses.Bytes())
}

//-----------------------------------------------------------------------------

// ValidatorsInfo represents the latest validator set, or the last height it changed
type ValidatorsInfo struct {
	ValidatorSet      *types.ValidatorSet
	LastHeightChanged int64
}

// Bytes serializes the ValidatorsInfo using go-amino.
func (valInfo *ValidatorsInfo) Bytes() []byte {
	return cdc.MustMarshalBinaryBare(valInfo)
}

// LoadValidators loads the ValidatorSet for a given height.
// Returns ErrNoValSetForHeight if the validator set can't be found for this height.
func LoadValidators(db dbm.DB, height int64) (*types.ValidatorSet, error) {
	valSet, _, err := LoadValidatorsWithStoredHeight(db, height)
	return valSet, err
}

// LoadValidators loads the ValidatorSet for a given height. plus the last LastHeightChanged
// Returns ErrNoValSetForHeight if the validator set can't be found for this height.
func LoadValidatorsWithStoredHeight(db dbm.DB, height int64) (*types.ValidatorSet, int64, error) {
	valInfo := loadValidatorsInfo(db, height)
	if valInfo == nil {
		return nil, -1, ErrNoValSetForHeight{height}
	}
	if valInfo.ValidatorSet == nil {
		lastStoredHeight := lastStoredHeightFor(height, valInfo.LastHeightChanged)
		valInfo2 := loadValidatorsInfo(db, lastStoredHeight)
		if valInfo2 == nil || valInfo2.ValidatorSet == nil {
			panic(
				fmt.Sprintf("Couldn't find validators at height %d (height %d was originally requested)",
					lastStoredHeight,
					height,
				),
			)
		}
		valInfo2.ValidatorSet.IncrementProposerPriority(int(height - lastStoredHeight)) // mutate
		valInfo = valInfo2
	}

	return valInfo.ValidatorSet, valInfo.LastHeightChanged, nil
}

func lastStoredHeightFor(height, lastHeightChanged int64) int64 {
	checkpointHeight := height - height%valSetCheckpointInterval
	return tmmath.MaxInt64(checkpointHeight, lastHeightChanged)
}

// CONTRACT: Returned ValidatorsInfo can be mutated.
func loadValidatorsInfo(db dbm.DB, height int64) *ValidatorsInfo {
	buf, err := db.Get(calcValidatorsKey(height))
	if err != nil {
		panic(err)
	}
	if len(buf) == 0 {
		return nil
	}

	v := new(ValidatorsInfo)
	err = cdc.UnmarshalBinaryBare(buf, v)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		tmos.Exit(fmt.Sprintf(`LoadValidators: Data has been corrupted or its spec has changed:
                %v\n`, err))
	}
	// TODO: ensure that buf is completely read.

	return v
}

// saveValidatorsInfo persists the validator set.
//
// `height` is the effective height for which the validator is responsible for
// signing. It should be called from s.Save(), right before the state itself is
// persisted.
func saveValidatorsInfo(db dbm.DB, height, lastHeightChanged int64, valSet *types.ValidatorSet) {
	if !IgnoreSmbCheck && lastHeightChanged > height {
		panic("LastHeightChanged cannot be greater than ValidatorsInfo height")
	}
	valInfo := &ValidatorsInfo{
		LastHeightChanged: lastHeightChanged,
	}
	// Only persist validator set if it was updated or checkpoint height (see
	// valSetCheckpointInterval) is reached.
	if height == lastHeightChanged || height%valSetCheckpointInterval == 0 {
		valInfo.ValidatorSet = valSet
	}
	db.Set(calcValidatorsKey(height), valInfo.Bytes())
}

//-----------------------------------------------------------------------------

// ConsensusParamsInfo represents the latest consensus params, or the last height it changed
type ConsensusParamsInfo struct {
	ConsensusParams   types.ConsensusParams
	LastHeightChanged int64
}

// Bytes serializes the ConsensusParamsInfo using go-amino.
func (params ConsensusParamsInfo) Bytes() []byte {
	return cdc.MustMarshalBinaryBare(params)
}

// LoadConsensusParams loads the ConsensusParams for a given height.
func LoadConsensusParams(db dbm.DB, height int64) (types.ConsensusParams, error) {
	empty := types.ConsensusParams{}

	paramsInfo := loadConsensusParamsInfo(db, height)
	if paramsInfo == nil {
		return empty, ErrNoConsensusParamsForHeight{height}
	}

	if paramsInfo.ConsensusParams.Equals(&empty) {
		paramsInfo2 := loadConsensusParamsInfo(db, paramsInfo.LastHeightChanged)
		if paramsInfo2 == nil {
			panic(
				fmt.Sprintf(
					"Couldn't find consensus params at height %d as last changed from height %d",
					paramsInfo.LastHeightChanged,
					height,
				),
			)
		}
		paramsInfo = paramsInfo2
	}

	return paramsInfo.ConsensusParams, nil
}

func loadConsensusParamsInfo(db dbm.DB, height int64) *ConsensusParamsInfo {
	buf, err := db.Get(calcConsensusParamsKey(height))
	if err != nil {
		panic(err)
	}
	if len(buf) == 0 {
		return nil
	}

	paramsInfo := new(ConsensusParamsInfo)
	err = cdc.UnmarshalBinaryBare(buf, paramsInfo)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		tmos.Exit(fmt.Sprintf(`LoadConsensusParams: Data has been corrupted or its spec has changed:
                %v\n`, err))
	}
	// TODO: ensure that buf is completely read.

	return paramsInfo
}

// saveConsensusParamsInfo persists the consensus params for the next block to disk.
// It should be called from s.Save(), right before the state itself is persisted.
// If the consensus params did not change after processing the latest block,
// only the last height for which they changed is persisted.
func saveConsensusParamsInfo(db dbm.DB, nextHeight, changeHeight int64, params types.ConsensusParams) {
	paramsInfo := &ConsensusParamsInfo{
		LastHeightChanged: changeHeight,
	}
	if changeHeight == nextHeight {
		paramsInfo.ConsensusParams = params
	}
	db.Set(calcConsensusParamsKey(nextHeight), paramsInfo.Bytes())
}
