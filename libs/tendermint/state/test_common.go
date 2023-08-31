package state

import (
	"bytes"
	"fmt"
	"time"

	"github.com/okx/brczero/libs/iavl"
	"github.com/okx/brczero/libs/tendermint/libs/log"

	dbm "github.com/okx/brczero/libs/tm-db"

	abci "github.com/okx/brczero/libs/tendermint/abci/types"
	"github.com/okx/brczero/libs/tendermint/crypto"
	"github.com/okx/brczero/libs/tendermint/crypto/ed25519"
	tmrand "github.com/okx/brczero/libs/tendermint/libs/rand"
	"github.com/okx/brczero/libs/tendermint/proxy"
	"github.com/okx/brczero/libs/tendermint/types"
	tmtime "github.com/okx/brczero/libs/tendermint/types/time"
)

var (
	chainID      = "execution_chain"
	testPartSize = 65536
	nTxsPerBlock = 10
)

type paramsChangeTestCase struct {
	height int64
	params types.ConsensusParams
}

// always returns true if asked if any evidence was already committed.
type mockEvPoolAlwaysCommitted struct{}

func (m mockEvPoolAlwaysCommitted) PendingEvidence(int64) []types.Evidence { return nil }
func (m mockEvPoolAlwaysCommitted) AddEvidence(types.Evidence) error       { return nil }
func (m mockEvPoolAlwaysCommitted) Update(*types.Block, State)             {}
func (m mockEvPoolAlwaysCommitted) IsCommitted(types.Evidence) bool        { return true }

func newTestApp() proxy.AppConns {
	app := &testApp{}
	cc := proxy.NewLocalClientCreator(app)
	return proxy.NewAppConns(cc)
}

func makeAndCommitGoodBlock(
	state State,
	height int64,
	lastCommit *types.Commit,
	proposerAddr []byte,
	blockExec *BlockExecutor,
	privVals map[string]types.PrivValidator,
	evidence []types.Evidence) (State, types.BlockID, *types.Commit, error) {
	// A good block passes
	state, blockID, err := makeAndApplyGoodBlock(state, height, lastCommit, proposerAddr, blockExec, evidence)
	if err != nil {
		return state, types.BlockID{}, nil, err
	}

	// Simulate a lastCommit for this block from all validators for the next height
	commit, err := makeValidCommit(height, blockID, state.Validators, privVals)
	if err != nil {
		return state, types.BlockID{}, nil, err
	}
	return state, blockID, commit, nil
}

func makeAndApplyGoodBlock(state State, height int64, lastCommit *types.Commit, proposerAddr []byte,
	blockExec *BlockExecutor, evidence []types.Evidence) (State, types.BlockID, error) {
	block, _ := state.MakeBlock(height, makeTxs(height), lastCommit, evidence, proposerAddr)
	if err := blockExec.ValidateBlock(state, block); err != nil {
		return state, types.BlockID{}, err
	}
	blockID := types.BlockID{Hash: block.Hash(),
		PartsHeader: types.PartSetHeader{Total: 3, Hash: tmrand.Bytes(32)}}
	state, _, err := blockExec.ApplyBlock(state, blockID, block)
	if err != nil {
		return state, types.BlockID{}, err
	}
	return state, blockID, nil
}

func makeValidCommit(
	height int64,
	blockID types.BlockID,
	vals *types.ValidatorSet,
	privVals map[string]types.PrivValidator,
) (*types.Commit, error) {
	sigs := make([]types.CommitSig, 0)
	for i := 0; i < vals.Size(); i++ {
		_, val := vals.GetByIndex(i)
		vote, err := types.MakeVote(height, blockID, vals, privVals[val.Address.String()], chainID, time.Now())
		if err != nil {
			return nil, err
		}
		sigs = append(sigs, vote.CommitSig())
	}
	return types.NewCommit(height, 0, blockID, sigs), nil
}

// make some bogus txs
func makeTxs(height int64) (txs []types.Tx) {
	for i := 0; i < nTxsPerBlock; i++ {
		txs = append(txs, types.Tx([]byte{byte(height), byte(i)}))
	}
	return txs
}

func makeState(nVals, height int) (State, dbm.DB, map[string]types.PrivValidator) {
	vals := make([]types.GenesisValidator, nVals)
	privVals := make(map[string]types.PrivValidator, nVals)
	for i := 0; i < nVals; i++ {
		secret := []byte(fmt.Sprintf("test%d", i))
		pk := ed25519.GenPrivKeyFromSecret(secret)
		valAddr := pk.PubKey().Address()
		vals[i] = types.GenesisValidator{
			Address: valAddr,
			PubKey:  pk.PubKey(),
			Power:   1000,
			Name:    fmt.Sprintf("test%d", i),
		}
		privVals[valAddr.String()] = types.NewMockPVWithParams(pk, false, false)
	}
	s, _ := MakeGenesisState(&types.GenesisDoc{
		ChainID:    chainID,
		Validators: vals,
		AppHash:    nil,
	})

	stateDB := dbm.NewMemDB()
	SaveState(stateDB, s)

	for i := 1; i < height; i++ {
		s.LastBlockHeight++
		s.LastValidators = s.Validators.Copy()
		SaveState(stateDB, s)
	}
	return s, stateDB, privVals
}

func makeBlock(state State, height int64) *types.Block {
	block, _ := state.MakeBlock(
		height,
		makeTxs(state.LastBlockHeight),
		new(types.Commit),
		nil,
		state.Validators.GetProposer().Address,
	)
	return block
}

func genValSet(size int) *types.ValidatorSet {
	vals := make([]*types.Validator, size)
	for i := 0; i < size; i++ {
		vals[i] = types.NewValidator(ed25519.GenPrivKey().PubKey(), 10)
	}
	return types.NewValidatorSet(vals)
}

func makeConsensusParams(
	blockBytes, blockGas int64,
	blockTimeIotaMs int64,
	evidenceAge int64,
) types.ConsensusParams {
	return types.ConsensusParams{
		Block: types.BlockParams{
			MaxBytes:   blockBytes,
			MaxGas:     blockGas,
			TimeIotaMs: blockTimeIotaMs,
		},
		Evidence: types.EvidenceParams{
			MaxAgeNumBlocks: evidenceAge,
			MaxAgeDuration:  time.Duration(evidenceAge),
		},
	}
}

func makeHeaderPartsResponsesValPubKeyChange(
	state State,
	pubkey crypto.PubKey,
) (types.Header, types.BlockID, *ABCIResponses) {

	block := makeBlock(state, state.LastBlockHeight+1)
	abciResponses := &ABCIResponses{
		EndBlock: &abci.ResponseEndBlock{ValidatorUpdates: nil},
	}

	// If the pubkey is new, remove the old and add the new.
	_, val := state.NextValidators.GetByIndex(0)
	if !bytes.Equal(pubkey.Bytes(), val.PubKey.Bytes()) {
		abciResponses.EndBlock = &abci.ResponseEndBlock{
			ValidatorUpdates: []abci.ValidatorUpdate{
				types.TM2PB.NewValidatorUpdate(val.PubKey, 0),
				types.TM2PB.NewValidatorUpdate(pubkey, 10),
			},
		}
	}

	return block.Header, types.BlockID{Hash: block.Hash(), PartsHeader: types.PartSetHeader{}}, abciResponses
}

func makeHeaderPartsResponsesValPowerChange(
	state State,
	power int64,
) (types.Header, types.BlockID, *ABCIResponses) {

	block := makeBlock(state, state.LastBlockHeight+1)
	abciResponses := &ABCIResponses{
		EndBlock: &abci.ResponseEndBlock{ValidatorUpdates: nil},
	}

	// If the pubkey is new, remove the old and add the new.
	_, val := state.NextValidators.GetByIndex(0)
	if val.VotingPower != power {
		abciResponses.EndBlock = &abci.ResponseEndBlock{
			ValidatorUpdates: []abci.ValidatorUpdate{
				types.TM2PB.NewValidatorUpdate(val.PubKey, power),
			},
		}
	}

	return block.Header, types.BlockID{Hash: block.Hash(), PartsHeader: types.PartSetHeader{}}, abciResponses
}

func makeHeaderPartsResponsesParams(
	state State,
	params types.ConsensusParams,
) (types.Header, types.BlockID, *ABCIResponses) {

	block := makeBlock(state, state.LastBlockHeight+1)
	abciResponses := &ABCIResponses{
		EndBlock: &abci.ResponseEndBlock{ConsensusParamUpdates: types.TM2PB.ConsensusParams(&params)},
	}
	return block.Header, types.BlockID{Hash: block.Hash(), PartsHeader: types.PartSetHeader{}}, abciResponses
}

func randomGenesisDoc() *types.GenesisDoc {
	pubkey := ed25519.GenPrivKey().PubKey()
	return &types.GenesisDoc{
		GenesisTime: tmtime.Now(),
		ChainID:     "abc",
		Validators: []types.GenesisValidator{
			{
				Address: pubkey.Address(),
				PubKey:  pubkey,
				Power:   10,
				Name:    "myval",
			},
		},
		ConsensusParams: types.DefaultConsensusParams(),
	}
}

//----------------------------------------------------------------------------

type testApp struct {
	abci.BaseApplication

	CommitVotes         []abci.VoteInfo
	ByzantineValidators []abci.Evidence
	ValidatorUpdates    []abci.ValidatorUpdate
}

var _ abci.Application = (*testApp)(nil)

func (app *testApp) Info(req abci.RequestInfo) (resInfo abci.ResponseInfo) {
	return abci.ResponseInfo{}
}

func (app *testApp) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	app.CommitVotes = req.LastCommitInfo.Votes
	app.ByzantineValidators = req.ByzantineValidators
	return abci.ResponseBeginBlock{}
}

func (app *testApp) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	return abci.ResponseEndBlock{ValidatorUpdates: app.ValidatorUpdates}
}

func (app *testApp) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	return abci.ResponseDeliverTx{Events: []abci.Event{}}
}

func (app *testApp) CheckTx(req abci.RequestCheckTx) abci.ResponseCheckTx {
	return abci.ResponseCheckTx{}
}

func (app *testApp) Commit(abci.RequestCommit) abci.ResponseCommit {
	return abci.ResponseCommit{}
}

func (app *testApp) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	return
}

//----------------------------------------------------------------------------------------------------
// Execute block without state. TODO: eliminate

// ExecCommitBlock executes and commits a block on the proxyApp without validating or mutating the state.
// It returns the application root hash (result of abci.Commit).
func ExecCommitBlock(
	appConnConsensus proxy.AppConnConsensus,
	block *types.Block,
	logger log.Logger,
	stateDB dbm.DB,
) ([]byte, error) {

	ctx := &executionTask{
		logger:   logger,
		block:    block,
		db:       stateDB,
		proxyApp: appConnConsensus,
	}

	_, err := execBlockOnProxyApp(ctx)
	if err != nil {
		logger.Error("Error executing block on proxy app", "height", block.Height, "err", err)
		return nil, err
	}
	// Commit block, get hash back
	res, err := appConnConsensus.CommitSync(abci.RequestCommit{})
	if err != nil {
		logger.Error("Client error during proxyAppConn.CommitSync", "err", res)
		return nil, err
	}
	// ResponseCommit has no error or log, just data
	return res.Data, nil
}

func execCommitBlockDelta(
	appConnConsensus proxy.AppConnConsensus,
	block *types.Block,
	logger log.Logger,
	stateDB dbm.DB,
) (*types.Deltas, []byte, error) {
	iavl.SetProduceDelta(true)
	types.UploadDelta = true
	deltas := &types.Deltas{Height: block.Height}

	ctx := &executionTask{
		logger:   logger,
		block:    block,
		db:       stateDB,
		proxyApp: appConnConsensus,
	}

	abciResponses, err := execBlockOnProxyApp(ctx)
	if err != nil {
		logger.Error("Error executing block on proxy app", "height", block.Height, "err", err)
		return nil, nil, err
	}
	abciResponsesBytes, err := types.Json.Marshal(abciResponses)
	if err != nil {
		return nil, nil, err
	}
	deltas.Payload.ABCIRsp = abciResponsesBytes

	// Commit block, get hash back
	res, err := appConnConsensus.CommitSync(abci.RequestCommit{})
	if err != nil {
		logger.Error("Client error during proxyAppConn.CommitSync", "err", res)
		return nil, nil, err
	}

	if res.DeltaMap != nil {
		deltaBytes, err := types.Json.Marshal(res.DeltaMap)
		if err != nil {
			return nil, nil, err
		}
		deltas.Payload.DeltasBytes = deltaBytes
		wdFunc := evmWatchDataManager.CreateWatchDataGenerator()
		if wd, err := wdFunc(); err == nil {
			deltas.Payload.WatchBytes = wd
		}
		wasmWdFunc := wasmWatchDataManager.CreateWatchDataGenerator()
		if wd, err := wasmWdFunc(); err == nil {
			deltas.Payload.WasmWatchBytes = wd
		}
	}

	// ResponseCommit has no error or log, just data
	return deltas, res.Data, nil
}
