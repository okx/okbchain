// nolint
// autogenerated code using github.com/rigelrozanski/multitool
// aliases generated for the following subdirectories:
package gov

import (
	"github.com/okx/okbchain/x/gov/keeper"
	"github.com/okx/okbchain/x/gov/types"
)

const (
	ModuleName        = types.ModuleName
	StoreKey          = types.StoreKey
	RouterKey         = types.RouterKey
	DefaultParamspace = types.DefaultParamspace
	ProposalTypeText  = types.ProposalTypeText
	QueryParams       = types.QueryParams

	StatusNil           = types.StatusNil
	StatusDepositPeriod = types.StatusDepositPeriod
	StatusVotingPeriod  = types.StatusVotingPeriod
	StatusPassed        = types.StatusPassed
	StatusRejected      = types.StatusRejected
	StatusFailed        = types.StatusFailed
)

var (
	// functions aliases
	RegisterCodec              = types.RegisterCodec
	RegisterProposalTypeCodec  = types.RegisterProposalTypeCodec
	ErrInvalidProposer         = types.ErrInvalidProposer
	ErrInvalidHeight           = types.ErrInvalidHeight
	ErrInvalidProposalContent  = types.ErrInvalidProposalContent
	ErrInvalidProposalType     = types.ErrInvalidProposalType
	ErrInvalidGenesis          = types.ErrInvalidGenesis
	ErrNoProposalHandlerExists = types.ErrNoProposalHandlerExists
	ProposalKey                = types.ProposalKey
	ActiveProposalByTimeKey    = types.ActiveProposalByTimeKey
	ActiveProposalQueueKey     = types.ActiveProposalQueueKey
	InactiveProposalByTimeKey  = types.InactiveProposalByTimeKey
	InactiveProposalQueueKey   = types.InactiveProposalQueueKey
	DepositKey                 = types.DepositKey
	VoteKey                    = types.VoteKey
	NewMsgSubmitProposal       = types.NewMsgSubmitProposal
	NewMsgDeposit              = types.NewMsgDeposit
	NewMsgVote                 = types.NewMsgVote
	ParamKeyTable              = types.ParamKeyTable
	NewDepositParams           = types.NewDepositParams
	NewTallyParams             = types.NewTallyParams
	NewVotingParams            = types.NewVotingParams
	NewParams                  = types.NewParams
	NewTallyResultFromMap      = types.NewTallyResultFromMap
	EmptyTallyResult           = types.EmptyTallyResult
	NewTextProposal            = types.NewTextProposal
	RegisterProposalType       = types.RegisterProposalType
	ContentFromProposalType    = types.ContentFromProposalType
	IsValidProposalType        = types.IsValidProposalType
	ProposalHandler            = types.ProposalHandler
	NewQueryProposalParams     = types.NewQueryProposalParams
	NewQueryDepositParams      = types.NewQueryDepositParams
	NewQueryVoteParams         = types.NewQueryVoteParams
	NewQueryProposalsParams    = types.NewQueryProposalsParams

	// variable aliases
	ModuleCdc                   = types.ModuleCdc
	ProposalsKeyPrefix          = types.ProposalsKeyPrefix
	ActiveProposalQueuePrefix   = types.ActiveProposalQueuePrefix
	InactiveProposalQueuePrefix = types.InactiveProposalQueuePrefix
	ProposalIDKey               = types.ProposalIDKey
	DepositsKeyPrefix           = types.DepositsKeyPrefix
	VotesKeyPrefix              = types.VotesKeyPrefix
	ParamStoreKeyDepositParams  = types.ParamStoreKeyDepositParams
	ParamStoreKeyVotingParams   = types.ParamStoreKeyVotingParams
	ParamStoreKeyTallyParams    = types.ParamStoreKeyTallyParams

	NewKeeper  = keeper.NewKeeper
	NewQuerier = keeper.NewQuerier
	NewRouter  = keeper.NewRouter
)

type (
	Content           = types.Content
	Handler           = types.Handler
	Deposit           = types.Deposit
	Deposits          = types.Deposits
	MsgSubmitProposal = types.MsgSubmitProposal
	MsgDeposit        = types.MsgDeposit
	MsgVote           = types.MsgVote
	DepositParams     = types.DepositParams
	TallyParams       = types.TallyParams
	VotingParams      = types.VotingParams
	Params            = types.Params
	Proposal          = types.Proposal
	Proposals         = types.Proposals
	ProposalStatus    = types.ProposalStatus
	TallyResult       = types.TallyResult
	Vote              = types.Vote
	Votes             = types.Votes
	Keeper            = keeper.Keeper
)
