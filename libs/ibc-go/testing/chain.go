package ibctesting

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/okx/okbchain/libs/tendermint/crypto"

	"github.com/okx/okbchain/libs/cosmos-sdk/client"
	types2 "github.com/okx/okbchain/libs/cosmos-sdk/codec/types"
	ibcmsg "github.com/okx/okbchain/libs/cosmos-sdk/types/ibc-adapter"
	ibc_tx "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/ibc-tx"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"

	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	//cryptotypes "github.com/okx/okbchain/libs/cosmos-sdk/crypto/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	authtypes "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"

	//banktypes "github.com/okx/okbchain/libs/cosmos-sdk/x/bank/types"
	capabilitykeeper "github.com/okx/okbchain/libs/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/okx/okbchain/libs/cosmos-sdk/x/capability/types"

	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	tmproto "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/crypto/tmhash"
	tmprototypes "github.com/okx/okbchain/libs/tendermint/proto/types"
	tmtypes "github.com/okx/okbchain/libs/tendermint/types"
	tmprotoversion "github.com/okx/okbchain/libs/tendermint/version"
	tmversion "github.com/okx/okbchain/libs/tendermint/version"
	stakingtypes "github.com/okx/okbchain/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/okx/okbchain/app/crypto/ethsecp256k1"
	apptypes "github.com/okx/okbchain/app/types"
	okcapptypes "github.com/okx/okbchain/app/types"
	clienttypes "github.com/okx/okbchain/libs/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/okx/okbchain/libs/ibc-go/modules/core/23-commitment/types"
	host "github.com/okx/okbchain/libs/ibc-go/modules/core/24-host"
	"github.com/okx/okbchain/libs/ibc-go/modules/core/exported"
	"github.com/okx/okbchain/libs/ibc-go/modules/core/types"
	ibctmtypes "github.com/okx/okbchain/libs/ibc-go/modules/light-clients/07-tendermint/types"
	"github.com/okx/okbchain/libs/ibc-go/testing/mock"
	"github.com/okx/okbchain/libs/ibc-go/testing/simapp"
)

type TestChainI interface {
	TxConfig() client.TxConfig
	T() *testing.T
	App() TestingApp
	GetContext() sdk.Context
	GetContextPointer() *sdk.Context
	GetClientState(clientID string) exported.ClientState
	QueryProof(key []byte) ([]byte, clienttypes.Height)
	QueryConsensusStateProof(clientID string) ([]byte, clienttypes.Height)
	GetConsensusState(clientID string, height exported.Height) (exported.ConsensusState, bool)
	GetPrefix() commitmenttypes.MerklePrefix
	LastHeader() *ibctmtypes.Header
	QueryServer() types.QueryService
	ChainID() string
	Codec() *codec.CodecProxy
	SenderAccount() sdk.Account
	SenderAccountPV() crypto.PrivKey
	SenderAccountPVBZ() []byte
	CurrentTMClientHeader() *ibctmtypes.Header
	ExpireClient(amount time.Duration)
	CurrentHeader() tmproto.Header
	CurrentHeaderTime(time.Time)
	NextBlock()
	BeginBlock()
	UpdateNextBlock()

	CreateTMClientHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, tmValSet, tmTrustedVals *tmtypes.ValidatorSet, signers []tmtypes.PrivValidator) *ibctmtypes.Header
	Vals() *tmtypes.ValidatorSet
	Signers() []tmtypes.PrivValidator
	GetSimApp() *simapp.SimApp
	GetChannelCapability(portID, channelID string) *capabilitytypes.Capability
	CreateChannelCapability(scopedKeeper capabilitykeeper.ScopedKeeper, portID, channelID string)
	SendMsgs(msgs ...ibcmsg.Msg) (*sdk.Result, error)
	QueryUpgradeProof(key []byte, height uint64) ([]byte, clienttypes.Height)
	Coordinator() *Coordinator
	QueryProofAtHeight(key []byte, height int64) ([]byte, clienttypes.Height)
	ConstructUpdateTMClientHeaderWithTrustedHeight(counterparty TestChainI, clientID string, trustedHeight clienttypes.Height) (*ibctmtypes.Header, error)
	ConstructUpdateTMClientHeader(counterparty TestChainI, clientID string) (*ibctmtypes.Header, error)
	sendMsgs(msgs ...ibcmsg.Msg) error
	GetValsAtHeight(height int64) (*tmtypes.ValidatorSet, bool)
	CreatePortCapability(scopedKeeper capabilitykeeper.ScopedKeeper, portID string)
	GetPortCapability(portID string) *capabilitytypes.Capability

	SenderAccounts() []SenderAccount
}

// TestChain is a testing struct that wraps a simapp with the last TM Header, the current ABCI
// header and the validators of the TestChain. It also contains a field called ChainID. This
// is the clientID that *other* chains use to refer to this TestChain. The SenderAccount
// is used for delivering transactions through the application state.
// NOTE: the actual application uses an empty chain-id for ease of testing.
type TestChain struct {
	t         *testing.T
	privKeyBz []byte
	context   sdk.Context

	coordinator   *Coordinator
	TApp          TestingApp
	chainID       string
	lastHeader    *ibctmtypes.Header // header for last block height committed
	currentHeader tmproto.Header     // header for current block height
	// QueryServer   types.QueryServer
	queryServer types.QueryService
	txConfig    client.TxConfig
	codec       *codec.CodecProxy

	vals    *tmtypes.ValidatorSet
	signers []tmtypes.PrivValidator

	senderPrivKey crypto.PrivKey
	senderAccount authtypes.Account

	senderAccounts []SenderAccount
}

var MaxAccounts = 10

type SenderAccount struct {
	SenderPrivKey crypto.PrivKey
	SenderAccount auth.Account
}

// NewTestChain initializes a new TestChain instance with a single validator set using a
// generated private key. It also creates a sender account to be used for delivering transactions.
//
// The first block height is committed to state in order to allow for client creations on
// counterparty chains. The TestChain will return with a block height starting at 2.
//
// Time management is handled by the Coordinator in order to ensure synchrony between chains.
// Each update of any chain increments the block header time for all chains by 5 seconds.
func NewTestChain(t *testing.T, coord *Coordinator, chainID string) TestChainI {
	// generate validator private/public key
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)

	senderAccs := []SenderAccount{}

	// generate genesis accounts
	for i := 0; i < MaxAccounts; i++ {
		senderPrivKey := secp256k1.GenPrivKey()
		i, ok := sdk.NewIntFromString("92233720368547758080")
		require.True(t, ok)
		balance := sdk.NewCoins(apptypes.NewPhotonCoin(i))

		acc := auth.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), balance, senderPrivKey.PubKey(), 0, 0)
		//amount, ok := sdk.NewIntFromString("10000000000000000000")
		//require.True(t, ok)

		// add sender account
		//balance := banktypes.Balance{
		//	Address: acc.GetAddress().String(),
		//	Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
		//}

		//genAccs = append(genAccs, acc)
		//genBals = append(genBals, balance)

		senderAcc := SenderAccount{
			SenderAccount: acc,
			SenderPrivKey: senderPrivKey,
		}

		senderAccs = append(senderAccs, senderAcc)
	}

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	signers := []tmtypes.PrivValidator{privVal}

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	//var pubkeyBytes secp256k1.PubKeySecp256k1
	//copy(pubkeyBytes[:], senderPrivKey.PubKey().Bytes())

	i, ok := sdk.NewIntFromString("92233720368547758080")
	require.True(t, ok)
	balance := sdk.NewCoins(apptypes.NewPhotonCoin(i))
	var genesisAcc authtypes.GenesisAccount
	genesisAcc = auth.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), balance, senderPrivKey.PubKey(), 0, 0)

	//amount, ok := sdk.NewIntFromString("10000000000000000000")
	//require.True(t, ok)

	//fromBalance := suite.App().AccountKeeper.GetAccount(suite.ctx, cmFrom).GetCoins()
	//var account *apptypes.EthAccount
	//balance = sdk.NewCoins(okexchaintypes.NewPhotonCoin(amount))
	//addr := sdk.AccAddress(pubKey.Address())
	//baseAcc := auth.NewBaseAccount(addr, balance, pubKey, 10, 50)
	//account = &apptypes.EthAccount{
	//	BaseAccount: baseAcc,
	//	CodeHash:    []byte{1, 2},
	//}
	//fmt.Println(account)
	//// balance := banktypes.Balance{
	//// 	Address: acc.GetAddress().String(),
	//// 	Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
	//// }

	app := SetupWithGenesisValSet(t, chainID, valSet, []authtypes.GenesisAccount{genesisAcc}, balance)

	// create current header and call begin block
	header := tmproto.Header{
		ChainID: chainID,
		Height:  1,
		Time:    coord.CurrentTime.UTC(),
	}

	txConfig := app.TxConfig()

	// create an account to send transactions from
	tchain := &TestChain{
		t:              t,
		privKeyBz:      senderPrivKey[:],
		coordinator:    coord,
		chainID:        chainID,
		TApp:           app,
		currentHeader:  header,
		queryServer:    app.GetIBCKeeper(),
		txConfig:       txConfig,
		codec:          app.AppCodec(),
		vals:           valSet,
		signers:        signers,
		senderPrivKey:  &senderPrivKey,
		senderAccount:  genesisAcc,
		senderAccounts: senderAccs,
	}

	//coord.UpdateNextBlock(tchain)
	coord.CommitBlock(tchain)
	//
	//coord.UpdateNextBlock(tchain)
	mockModuleAcc := tchain.GetSimApp().SupplyKeeper.GetModuleAccount(tchain.GetContext(), mock.ModuleName)
	require.NotNil(t, mockModuleAcc)

	return tchain
}

func NewTestEthChain(t *testing.T, coord *Coordinator, chainID string) *TestChain {
	// generate validator private/public key
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	signers := []tmtypes.PrivValidator{privVal}

	//Kb := keys.NewInMemory(hd.EthSecp256k1Options()...)
	// generate genesis account
	//info, err = Kb.CreateAccount(name, mnemonic, "", passWd, hdPath, hd.EthSecp256k1)
	senderPrivKey, _ := ethsecp256k1.GenerateKey() //secp256k1.GenPrivKey()

	ethPubkey := senderPrivKey.PubKey() //ethsecp256k1.PrivKey(senderPrivKey.Bytes()).PubKey()

	i, ok := sdk.NewIntFromString("92233720368547758080")
	require.True(t, ok)
	balance := sdk.NewCoins(apptypes.NewPhotonCoin(i))

	genesisAcc := &okcapptypes.EthAccount{
		BaseAccount: auth.NewBaseAccount(ethPubkey.Address().Bytes(), balance, ethPubkey, 0, 0),
		CodeHash:    []byte{},
	}
	//
	//senderPrivKey.PubKey().Address().Bytes()

	app := SetupWithGenesisValSet(t, chainID, valSet, []authtypes.GenesisAccount{genesisAcc}, balance)

	// create current header and call begin block
	header := tmproto.Header{
		ChainID: chainID,
		Height:  1,
		Time:    coord.CurrentTime.UTC(),
	}

	txConfig := app.TxConfig()

	// create an account to send transactions from
	return &TestChain{
		t:             t,
		coordinator:   coord,
		chainID:       chainID,
		TApp:          app,
		currentHeader: header,
		queryServer:   app.GetIBCKeeper(),
		txConfig:      txConfig,
		codec:         app.AppCodec(),
		vals:          valSet,
		signers:       signers,
		senderPrivKey: &senderPrivKey,
		senderAccount: genesisAcc,
		privKeyBz:     senderPrivKey[:],
	}

	//coord.UpdateNextBlock(tchain)
	//coord.CommitBlock(tchain)
	//
	//coord.UpdateNextBlock(tchain)

}
func (chain *TestChain) SenderAccounts() []SenderAccount {
	return chain.senderAccounts
}

// GetContext returns the current context for the application.
func (chain *TestChain) GetContext() sdk.Context {
	return chain.App().GetBaseApp().NewContext(false, chain.CurrentHeader())
}

func (chain *TestChain) TxConfig() client.TxConfig {
	interfaceRegistry := types2.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	chain.txConfig = ibc_tx.NewTxConfig(marshaler, ibc_tx.DefaultSignModes)
	return chain.txConfig
}

// GetSimApp returns the SimApp to allow usage ofnon-interface fields.
// CONTRACT: This function should not be called by third parties implementing
// their own SimApp.
func (chain *TestChain) GetSimApp() *simapp.SimApp {
	app, ok := chain.TApp.(*simapp.SimApp)
	require.True(chain.t, ok)

	return app
}

// QueryProof performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	return chain.QueryProofAtHeight(key, chain.App().LastBlockHeight())
}

// QueryProof performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) QueryProofAtHeight(key []byte, height int64) ([]byte, clienttypes.Height) {
	res := chain.App().Query(abci.RequestQuery{
		Path:   fmt.Sprintf("store/%s/key", host.StoreKey),
		Height: height - 1,
		Data:   key,
		Prove:  true,
	})

	merkleProof, err := commitmenttypes.ConvertProofs(res.GetProof())
	require.NoError(chain.t, err)

	proof, err := chain.App().AppCodec().GetProtocMarshal().MarshalBinaryBare(&merkleProof)
	require.NoError(chain.t, err)

	revision := clienttypes.ParseChainID(chain.ChainID())

	// proof height + 1 is returned as the proof created corresponds to the height the proof
	// was created in the IAVL tree. Tendermint and subsequently the clients that rely on it
	// have heights 1 above the IAVL tree. Thus we return proof height + 1
	return proof, clienttypes.NewHeight(revision, uint64(res.Height)+1)
}

// QueryUpgradeProof performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) QueryUpgradeProof(key []byte, height uint64) ([]byte, clienttypes.Height) {
	res := chain.App().Query(abci.RequestQuery{
		Path:   "store/upgrade/key",
		Height: int64(height - 1),
		Data:   key,
		Prove:  true,
	})

	//	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	merkleProof, err := commitmenttypes.ConvertProofs(res.GetProof())
	require.NoError(chain.t, err)

	// proof, err := chain.App().AppCodec().Marshal(&merkleProof)
	// require.NoError(chain.t, err)
	proof, err := chain.App().AppCodec().GetProtocMarshal().MarshalBinaryBare(&merkleProof)
	require.NoError(chain.t, err)

	revision := clienttypes.ParseChainID(chain.ChainID())

	// proof height + 1 is returned as the proof created corresponds to the height the proof
	// was created in the IAVL tree. Tendermint and subsequently the clients that rely on it
	// have heights 1 above the IAVL tree. Thus we return proof height + 1
	return proof, clienttypes.NewHeight(revision, uint64(res.Height+1))
}

// QueryConsensusStateProof performs an abci query for a consensus state
// stored on the given clientID. The proof and consensusHeight are returned.
func (chain *TestChain) QueryConsensusStateProof(clientID string) ([]byte, clienttypes.Height) {
	clientState := chain.GetClientState(clientID)

	consensusHeight := clientState.GetLatestHeight().(clienttypes.Height)
	consensusKey := host.FullConsensusStateKey(clientID, consensusHeight)
	proofConsensus, _ := chain.QueryProof(consensusKey)

	return proofConsensus, consensusHeight
}

// NextBlock sets the last header to the current header and increments the current header to be
// at the next block height. It does not update the time as that is handled by the Coordinator.
//
// CONTRACT: this function must only be called after app.Commit() occurs
func (chain *TestChain) NextBlock() {
	// set the last header to the current header
	// use nil trusted fields
	chain.SetLastHeader(chain.CurrentTMClientHeader())

	// increment the current header
	chain.SetCurrentHeader(tmproto.Header{
		ChainID: chain.ChainID(),
		Height:  chain.App().LastBlockHeight() + 1,
		AppHash: chain.App().LastCommitID().Hash,
		// NOTE: the time is increased by the coordinator to maintain time synchrony amongst
		// chains.
		Time:               chain.CurrentHeader().Time,
		ValidatorsHash:     chain.Vals().Hash(chain.App().LastBlockHeight() + 1),
		NextValidatorsHash: chain.Vals().Hash(chain.App().LastBlockHeight() + 1),
	})

	chain.App().BeginBlock(abci.RequestBeginBlock{Header: chain.CurrentHeader()})
}

func (chain *TestChain) BeginBlock() {
	chain.App().BeginBlock(abci.RequestBeginBlock{Header: chain.CurrentHeader()})
}

func (chain *TestChain) UpdateNextBlock() {
	chain.SetLastHeader(chain.CurrentTMClientHeader())

	// increment the current header
	chain.SetCurrentHeader(tmproto.Header{
		ChainID: chain.ChainID(),
		Height:  chain.App().LastBlockHeight() + 1,
		AppHash: chain.App().LastCommitID().Hash,
		// NOTE: the time is increased by the coordinator to maintain time synchrony amongst
		// chains.
		Time:               chain.CurrentHeader().Time,
		ValidatorsHash:     chain.Vals().Hash(chain.App().LastBlockHeight() + 1),
		NextValidatorsHash: chain.Vals().Hash(chain.App().LastBlockHeight() + 1),
	})
	chain.App().BeginBlock(abci.RequestBeginBlock{Header: chain.CurrentHeader()})
}

// sendMsgs delivers a transaction through the application without returning the result.
func (chain *TestChain) sendMsgs(msgs ...ibcmsg.Msg) error {
	_, err := chain.SendMsgs(msgs...)
	return err
}

// SendMsgs delivers a transaction through the application. It updates the senders sequence
// number and updates the TestChain's headers. It returns the result and error if one
// occurred.
func (chain *TestChain) SendMsgs(msgs ...ibcmsg.Msg) (*sdk.Result, error) {

	// ensure the chain has the latest time
	chain.Coordinator().UpdateTimeForChain(chain)
	_, r, err := simapp.SignAndDeliver(
		chain.t,
		chain.TxConfig(),
		chain.App().GetBaseApp(),
		//chain.GetContextPointer().BlockHeader(),
		chain.CurrentHeader(),
		msgs,
		chain.ChainID(),
		[]uint64{chain.SenderAccount().GetAccountNumber()},
		[]uint64{chain.SenderAccount().GetSequence()},
		true, true, chain.senderPrivKey,
	)
	if err != nil {
		return nil, err
	}

	// SignAndDeliver calls app.Commit()
	chain.NextBlock()

	// increment sequence for successful transaction execution
	chain.SenderAccount().SetSequence(chain.SenderAccount().GetSequence() + 1)

	chain.Coordinator().IncrementTime()

	return r, nil
}

// GetClientState retrieves the client state for the provided clientID. The client is
// expected to exist otherwise testing will fail.
func (chain *TestChain) GetClientState(clientID string) exported.ClientState {
	clientState, found := chain.App().GetIBCKeeper().ClientKeeper.GetClientState(chain.GetContext(), clientID)
	require.True(chain.t, found)

	return clientState
}

// GetConsensusState retrieves the consensus state for the provided clientID and height.
// It will return a success boolean depending on if consensus state exists or not.
func (chain *TestChain) GetConsensusState(clientID string, height exported.Height) (exported.ConsensusState, bool) {
	return chain.App().GetIBCKeeper().ClientKeeper.GetClientConsensusState(chain.GetContext(), clientID, height)
}

// GetValsAtHeight will return the validator set of the chain at a given height. It will return
// a success boolean depending on if the validator set exists or not at that height.
func (chain *TestChain) GetValsAtHeight(height int64) (*tmtypes.ValidatorSet, bool) {
	histInfo, ok := chain.App().GetStakingKeeper().GetHistoricalInfo(chain.GetContext(), height)
	if !ok {
		return nil, false
	}

	valSet := stakingtypes.Validators(histInfo.ValSet)

	validators := make([]*tmtypes.Validator, len(valSet))
	for i, val := range valSet {
		validators[i] = tmtypes.NewValidator(val.GetConsPubKey(), 1)
	}

	return tmtypes.NewValidatorSet(validators), true
}

// GetAcknowledgement retrieves an acknowledgement for the provided packet. If the
// acknowledgement does not exist then testing will fail.
func (chain *TestChain) GetAcknowledgement(packet exported.PacketI) []byte {
	ack, found := chain.App().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(chain.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	require.True(chain.t, found)

	return ack
}

// GetPrefix returns the prefix for used by a chain in connection creation
func (chain *TestChain) GetPrefix() commitmenttypes.MerklePrefix {
	return commitmenttypes.NewMerklePrefix(chain.App().GetIBCKeeper().ConnectionKeeper.GetCommitmentPrefix().Bytes())
}

// ConstructUpdateTMClientHeader will construct a valid 07-tendermint Header to update the
// light client on the source chain.
func (chain *TestChain) ConstructUpdateTMClientHeader(counterparty TestChainI, clientID string) (*ibctmtypes.Header, error) {
	return chain.ConstructUpdateTMClientHeaderWithTrustedHeight(counterparty, clientID, clienttypes.ZeroHeight())
}

// ConstructUpdateTMClientHeader will construct a valid 07-tendermint Header to update the
// light client on the source chain.
func (chain *TestChain) ConstructUpdateTMClientHeaderWithTrustedHeight(counterparty TestChainI, clientID string, trustedHeight clienttypes.Height) (*ibctmtypes.Header, error) {
	header := counterparty.LastHeader()
	// Relayer must query for LatestHeight on client to get TrustedHeight if the trusted height is not set
	if trustedHeight.IsZero() {
		trustedHeight = chain.GetClientState(clientID).GetLatestHeight().(clienttypes.Height)
	}
	var (
		tmTrustedVals *tmtypes.ValidatorSet
		ok            bool
	)
	// Once we get TrustedHeight from client, we must query the validators from the counterparty chain
	// If the LatestHeight == LastHeader.Height, then TrustedValidators are current validators
	// If LatestHeight < LastHeader.Height, we can query the historical validator set from HistoricalInfo
	if trustedHeight == counterparty.LastHeader().GetHeight() {
		tmTrustedVals = counterparty.Vals()
	} else {
		// NOTE: We need to get validators from counterparty at height: trustedHeight+1
		// since the last trusted validators for a header at height h
		// is the NextValidators at h+1 committed to in header h by
		// NextValidatorsHash
		tmTrustedVals, ok = counterparty.GetValsAtHeight(int64(trustedHeight.RevisionHeight + 1))
		if !ok {
			return nil, sdkerrors.Wrapf(ibctmtypes.ErrInvalidHeaderHeight, "could not retrieve trusted validators at trustedHeight: %d", trustedHeight)
		}
	}
	// inject trusted fields into last header
	// for now assume revision number is 0
	header.TrustedHeight = trustedHeight

	trustedVals, err := tmTrustedVals.ToProto()
	if err != nil {
		return nil, err
	}
	header.TrustedValidators = trustedVals

	return header, nil

}

// ExpireClient fast forwards the chain's block time by the provided amount of time which will
// expire any clients with a trusting period less than or equal to this amount of time.
func (chain *TestChain) ExpireClient(amount time.Duration) {
	chain.Coordinator().IncrementTimeBy(amount)
}

// CurrentTMClientHeader creates a TM header using the current header parameters
// on the chain. The trusted fields in the header are set to nil.
func (chain *TestChain) CurrentTMClientHeader() *ibctmtypes.Header {
	return chain.CreateTMClientHeader(chain.chainID, chain.CurrentHeader().Height, clienttypes.Height{}, chain.CurrentHeader().Time, chain.Vals(), nil, chain.Signers())
}

// CreateTMClientHeader creates a TM header to update the TM client. Args are passed in to allow
// caller flexibility to use params that differ from the chain.
func (chain *TestChain) CreateTMClientHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, tmValSet, tmTrustedVals *tmtypes.ValidatorSet, signers []tmtypes.PrivValidator) *ibctmtypes.Header {
	var (
		valSet      *tmprototypes.ValidatorSet
		trustedVals *tmprototypes.ValidatorSet
	)
	require.NotNil(chain.t, tmValSet)

	vsetHash := tmValSet.Hash(blockHeight)

	tmHeader := tmtypes.Header{
		Version:            tmprotoversion.Consensus{Block: tmversion.BlockProtocol, App: 2},
		ChainID:            chainID,
		Height:             blockHeight,
		Time:               timestamp,
		LastBlockID:        MakeBlockID(make([]byte, tmhash.Size), 10_000, make([]byte, tmhash.Size)),
		LastCommitHash:     chain.App().LastCommitID().Hash,
		DataHash:           tmhash.Sum([]byte("data_hash")),
		ValidatorsHash:     vsetHash,
		NextValidatorsHash: vsetHash,
		ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
		AppHash:            chain.CurrentHeader().AppHash,
		LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
		EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
		ProposerAddress:    tmValSet.Proposer.Address, //nolint:staticcheck
	}
	hhash := tmHeader.Hash()
	blockID := MakeBlockID(hhash, 3, tmhash.Sum([]byte("part_set")))
	voteSet := tmtypes.NewVoteSet(chainID, blockHeight, 1, tmtypes.PrecommitType, tmValSet)

	commit, err := tmtypes.MakeCommit(blockID, blockHeight, 1, voteSet, signers, timestamp)
	require.NoError(chain.t, err)

	signedHeader := &tmtypes.SignedHeader{
		Header: &tmHeader,
		Commit: commit,
	}

	if tmValSet != nil {
		valSet, err = tmValSet.ToProto()
		if err != nil {
			panic(err)
		}
	}

	if tmTrustedVals != nil {
		trustedVals, err = tmTrustedVals.ToProto()
		if err != nil {
			panic(err)
		}
	}

	// The trusted fields may be nil. They may be filled before relaying messages to a client.
	// The relayer is responsible for querying client and injecting appropriate trusted fields.
	return &ibctmtypes.Header{
		SignedHeader:      signedHeader.ToProto(),
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedVals,
	}
}

// MakeBlockID copied unimported test functions from tmtypes to use them here
func MakeBlockID(hash []byte, partSetSize uint32, partSetHash []byte) tmtypes.BlockID {
	return tmtypes.BlockID{
		Hash: hash,
		PartsHeader: tmtypes.PartSetHeader{
			Total: int(partSetSize),
			Hash:  partSetHash,
		},
	}
}

// CreateSortedSignerArray takes two PrivValidators, and the corresponding Validator structs
// (including voting power). It returns a signer array of PrivValidators that matches the
// sorting of ValidatorSet.
// The sorting is first by .VotingPower (descending), with secondary index of .Address (ascending).
func CreateSortedSignerArray(altPrivVal, suitePrivVal tmtypes.PrivValidator,
	altVal, suiteVal *tmtypes.Validator) []tmtypes.PrivValidator {

	switch {
	case altVal.VotingPower > suiteVal.VotingPower:
		return []tmtypes.PrivValidator{altPrivVal, suitePrivVal}
	case altVal.VotingPower < suiteVal.VotingPower:
		return []tmtypes.PrivValidator{suitePrivVal, altPrivVal}
	default:
		if bytes.Compare(altVal.Address, suiteVal.Address) == -1 {
			return []tmtypes.PrivValidator{altPrivVal, suitePrivVal}
		}
		return []tmtypes.PrivValidator{suitePrivVal, altPrivVal}
	}
}

// CreatePortCapability binds and claims a capability for the given portID if it does not
// already exist. This function will fail testing on any resulting error.
// NOTE: only creation of a capbility for a transfer or mock port is supported
// Other applications must bind to the port in InitGenesis or modify this code.
func (chain *TestChain) CreatePortCapability(scopedKeeper capabilitykeeper.ScopedKeeper, portID string) {
	// check if the portId is already binded, if not bind it
	_, ok := chain.App().GetScopedIBCKeeper().GetCapability(chain.GetContext(), host.PortPath(portID))
	if !ok {
		// create capability using the IBC capability keeper
		cap, err := chain.App().GetScopedIBCKeeper().NewCapability(chain.GetContext(), host.PortPath(portID))
		require.NoError(chain.t, err)

		// claim capability using the scopedKeeper
		err = scopedKeeper.ClaimCapability(chain.GetContext(), cap, host.PortPath(portID))
		require.NoError(chain.t, err)
	}

	chain.App().Commit(abci.RequestCommit{})

	chain.NextBlock()
}

// GetPortCapability returns the port capability for the given portID. The capability must
// exist, otherwise testing will fail.
func (chain *TestChain) GetPortCapability(portID string) *capabilitytypes.Capability {
	cap, ok := chain.App().GetScopedIBCKeeper().GetCapability(chain.GetContext(), host.PortPath(portID))
	require.True(chain.t, ok)

	return cap
}

// CreateChannelCapability binds and claims a capability for the given portID and channelID
// if it does not already exist. This function will fail testing on any resulting error. The
// scoped keeper passed in will claim the new capability.
func (chain *TestChain) CreateChannelCapability(scopedKeeper capabilitykeeper.ScopedKeeper, portID, channelID string) {
	capName := host.ChannelCapabilityPath(portID, channelID)
	// check if the portId is already binded, if not bind it
	_, ok := chain.App().GetScopedIBCKeeper().GetCapability(chain.GetContext(), capName)
	if !ok {
		cap, err := chain.App().GetScopedIBCKeeper().NewCapability(chain.GetContext(), capName)
		require.NoError(chain.t, err)
		err = scopedKeeper.ClaimCapability(chain.GetContext(), cap, capName)
		require.NoError(chain.t, err)
	}

	chain.App().Commit(abci.RequestCommit{})

	chain.NextBlock()
}

// GetChannelCapability returns the channel capability for the given portID and channelID.
// The capability must exist, otherwise testing will fail.
func (chain *TestChain) GetChannelCapability(portID, channelID string) *capabilitytypes.Capability {
	cap, ok := chain.App().GetScopedIBCKeeper().GetCapability(chain.GetContext(), host.ChannelCapabilityPath(portID, channelID))
	require.True(chain.t, ok)

	return cap
}

// implement
func (chain *TestChain) T() *testing.T {
	return chain.t
}
func (chain *TestChain) App() TestingApp {
	return chain.GetSimApp()
}

func (chain *TestChain) GetContextPointer() *sdk.Context {
	return &chain.context
}

// func (chain *TestChain) QueryProof(key []byte) ([]byte, clienttypes.Height)  {}
// func (chain *TestChain) GetConsensusState(clientID string, height exported.Height) (exported.ConsensusState, bool) {
// }
// func (chain *TestChain) GetPrefix() commitmenttypes.MerklePrefix   {}
func (chain *TestChain) LastHeader() *ibctmtypes.Header {
	return chain.lastHeader
}
func (chain *TestChain) SetLastHeader(lh *ibctmtypes.Header) {
	chain.lastHeader = lh
}

func (chain *TestChain) QueryServer() types.QueryService {
	return chain.queryServer
}
func (chain *TestChain) ChainID() string {
	return chain.chainID
}

func (chain *TestChain) Codec() *codec.CodecProxy {
	return chain.codec
}
func (chain *TestChain) SenderAccount() sdk.Account {
	return chain.senderAccount
}
func (chain *TestChain) SenderAccountPV() crypto.PrivKey {

	return chain.senderPrivKey
}

func (chain *TestChain) SenderAccountPVBZ() []byte {
	return chain.privKeyBz
}

// func (chain *TestChain) CurrentTMClientHeader() *ibctmtypes.Header {}
func (chain *TestChain) CurrentHeader() tmproto.Header {
	return chain.currentHeader
}
func (chain *TestChain) SetCurrentHeader(h tmproto.Header) {
	chain.currentHeader = h
}

// func (chain *TestChain) NextBlock()                                {}
//
// func CreateTMClientHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, tmValSet, tmTrustedVals *tmtypes.ValidatorSet, signers []tmtypes.PrivValidator) *ibctmtypes.Header {
// }
func (chain *TestChain) Vals() *tmtypes.ValidatorSet {
	return chain.vals
}

func (chain *TestChain) Signers() []tmtypes.PrivValidator {
	return chain.signers
}

// func GetSimApp() *simapp.SimApp                                                                    {}
// func GetChannelCapability(portID, channelID string) *capabilitytypes.Capability                    {}
// func CreateChannelCapability(scopedKeeper capabilitykeeper.ScopedKeeper, portID, channelID string) {}
// func SendMsgs(msgs ...sdk.Msg) (*sdk.Result, error)                                                {}
// func QueryUpgradeProof(key []byte, height uint64) ([]byte, clienttypes.Height)                     {}
func (chain *TestChain) Coordinator() *Coordinator {
	return chain.coordinator
}

func (chain *TestChain) CurrentHeaderTime(t time.Time) {
	chain.currentHeader.Time = t
}

//func QueryProofAtHeight(key []byte, height uint64) ([]byte, clienttypes.Height) {}
