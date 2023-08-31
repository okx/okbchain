package types_test

import (
	"testing"
	"time"

	tmproto "github.com/okx/brczero/libs/tendermint/abci/types"
	tmbytes "github.com/okx/brczero/libs/tendermint/libs/bytes"
	tmtypes "github.com/okx/brczero/libs/tendermint/types"
	"github.com/stretchr/testify/suite"

	"github.com/okx/brczero/libs/cosmos-sdk/codec"
	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
	clienttypes "github.com/okx/brczero/libs/ibc-go/modules/core/02-client/types"
	ibctmtypes "github.com/okx/brczero/libs/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/okx/brczero/libs/ibc-go/testing"
	ibctestingmock "github.com/okx/brczero/libs/ibc-go/testing/mock"
	"github.com/okx/brczero/libs/ibc-go/testing/simapp"
)

const (
	chainID                          = "gaia"
	chainIDRevision0                 = "gaia-revision-0"
	chainIDRevision1                 = "gaia-revision-1"
	clientID                         = "gaiamainnet"
	trustingPeriod     time.Duration = time.Hour * 24 * 7 * 2
	trustingPeriod_one time.Duration = time.Hour * 24 * 7 * 1
	ubdPeriod          time.Duration = time.Hour * 24 * 7 * 3
	maxClockDrift      time.Duration = time.Second * 10
)

var (
	height          = clienttypes.NewHeight(0, 4)
	newClientHeight = clienttypes.NewHeight(1, 1)
	upgradePath     = []string{"upgrade", "upgradedIBCState"}
)

type TendermintTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA ibctesting.TestChainI
	chainB ibctesting.TestChainI

	// TODO: deprecate usage in favor of testing package
	ctx        sdk.Context
	cdc        *codec.CodecProxy
	privVal    tmtypes.PrivValidator
	valSet     *tmtypes.ValidatorSet
	valsHash   tmbytes.HexBytes
	header     *ibctmtypes.Header
	now        time.Time
	headerTime time.Time
	clientTime time.Time
}

func (suite *TendermintTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)

	// TODO: deprecate usage in favor of testing package
	checkTx := false
	app := simapp.Setup(checkTx)

	suite.cdc = app.AppCodec()

	// now is the time of the current chain, must be after the updating header
	// mocks ctx.BlockTime()
	suite.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	suite.clientTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	// Header time is intended to be time for any new header used for updates
	suite.headerTime = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	suite.privVal = ibctestingmock.NewPV()

	pubKey, err := suite.privVal.GetPubKey()
	suite.Require().NoError(err)

	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	val := tmtypes.NewValidator(pubKey, 10)
	suite.valSet = tmtypes.NewValidatorSet([]*tmtypes.Validator{val})
	suite.valsHash = suite.valSet.Hash(1)
	suite.header = suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, suite.valSet, suite.valSet, []tmtypes.PrivValidator{suite.privVal})
	suite.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1, Time: suite.now})
}

func getSuiteSigners(suite *TendermintTestSuite) []tmtypes.PrivValidator {
	return []tmtypes.PrivValidator{suite.privVal}
}

func getBothSigners(suite *TendermintTestSuite, altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) (*tmtypes.ValidatorSet, []tmtypes.PrivValidator) {
	// Create bothValSet with both suite validator and altVal. Would be valid update
	bothValSet := tmtypes.NewValidatorSet(append(suite.valSet.Validators, altVal))
	// Create signer array and ensure it is in same order as bothValSet
	_, suiteVal := suite.valSet.GetByIndex(0)
	bothSigners := ibctesting.CreateSortedSignerArray(altPrivVal, suite.privVal, altVal, suiteVal)
	return bothValSet, bothSigners
}

func TestTendermintTestSuite(t *testing.T) {
	suite.Run(t, new(TendermintTestSuite))
}
