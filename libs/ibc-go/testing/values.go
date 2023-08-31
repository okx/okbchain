/*
This file contains the variables, constants, and default values
used in the testing package and commonly defined in tests.
*/
package ibctesting

import (
	"strconv"
	"time"

	ibcfeetypes "github.com/okx/brczero/libs/ibc-go/modules/apps/29-fee/types"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"

	ibctransfertypes "github.com/okx/brczero/libs/ibc-go/modules/apps/transfer/types"
	connectiontypes "github.com/okx/brczero/libs/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/okx/brczero/libs/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/okx/brczero/libs/ibc-go/modules/core/23-commitment/types"
	ibctmtypes "github.com/okx/brczero/libs/ibc-go/modules/light-clients/07-tendermint/types"
	"github.com/okx/brczero/libs/ibc-go/testing/mock"
)

const (
	FirstClientID     = "07-tendermint-0"
	FirstChannelID    = "channel-0"
	FirstConnectionID = "connection-0"

	// Default params constants used to create a TM client
	TrustingPeriod     time.Duration = time.Hour * 24 * 7 * 1
	UnbondingPeriod    time.Duration = time.Hour * 24 * 7 * 2
	MaxClockDrift      time.Duration = time.Second * 10
	DefaultDelayPeriod uint64        = 0

	DefaultChannelVersion = ibctransfertypes.Version
	InvalidID             = "IDisInvalid"

	// Application Ports
	TransferPort = ibctransfertypes.ModuleName
	MockPort     = mock.ModuleName

	// used for testing proposals
	Title       = "title"
	Description = "description"

	LongString = "LoremipsumdolorsitameconsecteturadipiscingeliseddoeiusmodtemporincididuntutlaboreetdoloremagnaaliquUtenimadminimveniamquisnostrudexercitationullamcolaborisnisiutaliquipexeacommodoconsequDuisauteiruredolorinreprehenderitinvoluptateelitsseillumoloreufugiatnullaariaturEcepteurintoccaectupidatatonroidentuntnulpauifficiaeseruntmollitanimidestlaborum"

	MockFeePort = mock.ModuleName + ibcfeetypes.ModuleName
)

var (
	DefaultOpenInitVersion *connectiontypes.Version

	// Default params variables used to create a TM client
	DefaultTrustLevel ibctmtypes.Fraction = ibctmtypes.DefaultTrustLevel
	TestCoin                              = sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))

	UpgradePath = []string{"upgrade", "upgradedIBCState"}

	ConnectionVersion = connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions())[0]

	MockAcknowledgement          = mock.MockAcknowledgement.Acknowledgement()
	MockPacketData               = mock.MockPacketData
	MockFailPacketData           = mock.MockFailPacketData
	MockRecvCanaryCapabilityName = mock.MockRecvCanaryCapabilityName

	prefix = commitmenttypes.NewMerklePrefix([]byte("ibc"))
)

func GetMockRecvCanaryCapabilityName(packet channeltypes.Packet) string {
	return MockRecvCanaryCapabilityName + strconv.Itoa(int(packet.GetSequence()))
}
