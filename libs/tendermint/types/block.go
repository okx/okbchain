package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	gogotypes "github.com/gogo/protobuf/types"
	"github.com/okx/okbchain/libs/system/trace"
	"github.com/okx/okbchain/libs/tendermint/libs/compress"
	tmtime "github.com/okx/okbchain/libs/tendermint/types/time"

	"github.com/tendermint/go-amino"

	"github.com/pkg/errors"

	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/merkle"
	"github.com/okx/okbchain/libs/tendermint/crypto/tmhash"
	"github.com/okx/okbchain/libs/tendermint/libs/bits"
	tmbytes "github.com/okx/okbchain/libs/tendermint/libs/bytes"
	tmmath "github.com/okx/okbchain/libs/tendermint/libs/math"
	tmproto "github.com/okx/okbchain/libs/tendermint/proto/types"
	tmversion "github.com/okx/okbchain/libs/tendermint/proto/version"
	"github.com/okx/okbchain/libs/tendermint/version"
)

const (
	// MaxHeaderBytes is a maximum header size (including amino overhead).
	MaxHeaderBytes int64 = 632

	// MaxAminoOverheadForBlock - maximum amino overhead to encode a block (up to
	// MaxBlockSizeBytes in size) not including it's parts except Data.
	// This means it also excludes the overhead for individual transactions.
	// To compute individual transactions' overhead use types.ComputeAminoOverhead(tx types.Tx, fieldNum int).
	//
	// Uvarint length of MaxBlockSizeBytes: 4 bytes
	// 2 fields (2 embedded):               2 bytes
	// Uvarint length of Data.Txs:          4 bytes
	// Data.Txs field:                      1 byte
	MaxAminoOverheadForBlock int64 = 11

	// CompressDividing is used to divide compressType and compressFlag of compressSign
	// the compressSign = CompressType * CompressDividing + CompressFlag
	CompressDividing int = 10

	FlagBlockCompressType      = "block-compress-type"
	FlagBlockCompressFlag      = "block-compress-flag"
	FlagBlockCompressThreshold = "block-compress-threshold"
)

var (
	BlockCompressType      = 0x00
	BlockCompressFlag      = 0
	BlockCompressThreshold = 1024000
)

type BlockExInfo struct {
	BlockCompressType int
	BlockCompressFlag int
	BlockPartSize     int
}

func (info BlockExInfo) IsCompressed() bool {
	return info.BlockCompressType != 0
}

// Block defines the atomic unit of a Tendermint blockchain.
type Block struct {
	mtx sync.Mutex

	Header     `json:"header"`
	Data       `json:"data"`
	Evidence   EvidenceData `json:"evidence"`
	LastCommit *Commit      `json:"last_commit"`
}

func (b *Block) AminoSize(cdc *amino.Codec) int {
	var size = 0

	headerSize := b.Header.AminoSize(cdc)
	if headerSize > 0 {
		size += 1 + amino.UvarintSize(uint64(headerSize)) + headerSize
	}

	dataSize := b.Data.AminoSize(cdc)
	if dataSize > 0 {
		size += 1 + amino.UvarintSize(uint64(dataSize)) + dataSize
	}

	evidenceSize := b.Evidence.AminoSize(cdc)
	if evidenceSize > 0 {
		size += 1 + amino.UvarintSize(uint64(evidenceSize)) + evidenceSize
	}

	if b.LastCommit != nil {
		commitSize := b.LastCommit.AminoSize(cdc)
		size += 1 + amino.UvarintSize(uint64(commitSize)) + commitSize
	}

	return size
}

func (b *Block) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]

		if len(data) == 0 {
			break
		}

		pos, aminoType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if aminoType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}

			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("not enough data to read field %d", pos)
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			err = b.Header.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		case 2:
			err = b.Data.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		case 3:
			err = b.Evidence.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		case 4:
			b.LastCommit = new(Commit)
			err = b.LastCommit.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

// ValidateBasic performs basic validation that doesn't involve state data.
// It checks the internal consistency of the block.
// Further validation is done using state#ValidateBlock.
func (b *Block) ValidateBasic() error {
	if b == nil {
		return errors.New("nil block")
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()

	if err := b.Header.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}

	// Validate the last commit and its hash.
	if b.Header.Height > GetStartBlockHeight()+1 {
		if b.LastCommit == nil {
			return errors.New("nil LastCommit")
		}
		if err := b.LastCommit.ValidateBasic(); err != nil {
			return fmt.Errorf("wrong LastCommit: %v", err)
		}
	}

	if !bytes.Equal(b.LastCommitHash, b.LastCommit.Hash()) {
		return fmt.Errorf("wrong Header.LastCommitHash. Expected %v, got %v",
			b.LastCommit.Hash(),
			b.LastCommitHash,
		)
	}

	// NOTE: b.Data.Txs may be nil, but b.Data.Hash() still works fine.
	if !bytes.Equal(b.DataHash, b.Data.Hash(b.Height)) {
		return fmt.Errorf(
			"wrong Header.DataHash. Expected %v, got %v",
			b.Data.Hash(b.Height),
			b.DataHash,
		)
	}

	// NOTE: b.Evidence.Evidence may be nil, but we're just looping.
	for i, ev := range b.Evidence.Evidence {
		if err := ev.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid evidence (#%d): %v", i, err)
		}
	}

	if !bytes.Equal(b.EvidenceHash, b.Evidence.Hash()) {
		return fmt.Errorf("wrong Header.EvidenceHash. Expected %v, got %v",
			b.EvidenceHash,
			b.Evidence.Hash(),
		)
	}

	return nil
}

// fillHeader fills in any remaining header fields that are a function of the block data
func (b *Block) fillHeader() {
	if b.LastCommitHash == nil {
		b.LastCommitHash = b.LastCommit.Hash()
	}
	if b.DataHash == nil {
		b.DataHash = b.Data.Hash(b.Height)
	}
	if b.EvidenceHash == nil {
		b.EvidenceHash = b.Evidence.Hash()
	}
}

// Hash computes and returns the block hash.
// If the block is incomplete, block hash is nil for safety.
func (b *Block) Hash() tmbytes.HexBytes {
	if b == nil {
		return nil
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.LastCommit == nil {
		return nil
	}
	b.fillHeader()

	return b.Header.Hash()
}

// MakePartSet returns a PartSet containing parts of a serialized block.
// This is the form in which the block is gossipped to peers.
// CONTRACT: partSize is greater than zero.
func (b *Block) MakePartSet(partSize int) *PartSet {
	return b.MakePartSetByExInfo(&BlockExInfo{
		BlockCompressType: BlockCompressType,
		BlockCompressFlag: BlockCompressFlag,
		BlockPartSize:     partSize,
	})
}

func (b *Block) MakePartSetByExInfo(exInfo *BlockExInfo) *PartSet {
	if b == nil {
		return nil
	}
	if exInfo == nil {
		exInfo = &BlockExInfo{
			BlockCompressType: BlockCompressType,
			BlockCompressFlag: BlockCompressFlag,
			BlockPartSize:     BlockPartSizeBytes,
		}
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()

	// We prefix the byte length, so that unmarshaling
	// can easily happen via a reader.
	bz, err := cdc.MarshalBinaryLengthPrefixed(b)
	if err != nil {
		panic(err)
	}

	payload := compressBlock(bz, exInfo.BlockCompressType, exInfo.BlockCompressFlag)

	return NewPartSetFromData(payload, exInfo.BlockPartSize)

}

func compressBlock(bz []byte, compressType, compressFlag int) []byte {
	if compressType == 0 || len(bz) <= BlockCompressThreshold {
		return bz
	}
	if compressType >= CompressDividing || compressFlag >= CompressDividing {
		// unsupported compressType or compressFlag
		return bz
	}

	t0 := tmtime.Now()
	cz, err := compress.Compress(compressType, compressFlag, bz)
	if err != nil {
		return bz
	}
	t1 := tmtime.Now()

	trace.GetElapsedInfo().AddInfo(trace.CompressBlock, fmt.Sprintf("%dms", t1.Sub(t0).Milliseconds()))
	// tell receiver which compress type and flag
	// tens digit is compressType and unit digit is compressFlag
	// compressSign: XY means, compressType: X, compressFlag: Y
	compressSign := compressType*CompressDividing + compressFlag
	return append(cz, byte(compressSign))
}

func UncompressBlockFromReader(pbpReader io.Reader) (io.Reader, error) {
	// received compressed block bytes, uncompress with the flag:Proposal.CompressBlock
	compressed, err := io.ReadAll(pbpReader)
	if err != nil {
		return nil, err
	}
	t0 := tmtime.Now()
	original, compressSign, err := UncompressBlockFromBytes(compressed)
	if err != nil {
		return nil, err
	}
	t1 := tmtime.Now()

	if compressSign != 0 {
		compressRatio := float64(len(compressed)) / float64(len(original))
		trace.GetElapsedInfo().AddInfo(trace.UncompressBlock, fmt.Sprintf("%.2f/%dms",
			compressRatio, t1.Sub(t0).Milliseconds()))
	}

	return bytes.NewBuffer(original), nil
}

// UncompressBlockFromBytes uncompress from compressBytes to blockPart bytes, and returns the compressSign
// compressSign contains compressType and compressFlag
// the compressSign: XY means, compressType: X, compressFlag: Y
func UncompressBlockFromBytes(payload []byte) (res []byte, compressSign int, err error) {
	var buf bytes.Buffer
	compressSign, err = UncompressBlockFromBytesTo(payload, &buf)
	if err == nil && compressSign == 0 {
		return payload, 0, nil
	}
	res = buf.Bytes()
	return
}

func IsBlockDataCompressed(payload []byte) bool {
	// try parse Uvarint to check if it is compressed
	compressBytesLen, n := binary.Uvarint(payload)
	if compressBytesLen == uint64(len(payload)-n) {
		return false
	} else {
		return true
	}
}

// UncompressBlockFromBytesTo uncompress payload to buf, and returns the compressSign,
// if payload is not compressed, compressSign will be 0, and buf will not be changed.
func UncompressBlockFromBytesTo(payload []byte, buf *bytes.Buffer) (compressSign int, err error) {
	if IsBlockDataCompressed(payload) {
		// the block has compressed and the last byte is compressSign
		compressSign = int(payload[len(payload)-1])
		err = compress.UnCompressTo(compressSign/CompressDividing, payload[:len(payload)-1], buf)
	}
	return
}

// HashesTo is a convenience function that checks if a block hashes to the given argument.
// Returns false if the block is nil or the hash is empty.
func (b *Block) HashesTo(hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	if b == nil {
		return false
	}
	return bytes.Equal(b.Hash(), hash)
}

// Size returns size of the block in bytes.
func (b *Block) Size() int {
	bz, err := cdc.MarshalBinaryBare(b)
	if err != nil {
		return 0
	}
	return len(bz)
}

// TODO : Replace the original logic of Size with the logic of FastSize

// FastSize returns size of the block in bytes. and more efficient than Size().
// But we can't make sure it's completely correct yet, when we're done testing, we'll replace Size with FastSize
func (b *Block) FastSize() (size int) {
	defer func() {
		if r := recover(); r != nil {
			size = 0
		}
	}()
	return b.AminoSize(cdc)
}

// String returns a string representation of the block
func (b *Block) String() string {
	return b.StringIndented("")
}

// StringIndented returns a string representation of the block
func (b *Block) StringIndented(indent string) string {
	if b == nil {
		return "nil-Block"
	}
	return fmt.Sprintf(`Block{
%s  %v
%s  %v
%s  %v
%s  %v
%s}#%v`,
		indent, b.Header.StringIndented(indent+"  "),
		indent, b.Data.StringIndented(indent+"  "),
		indent, b.Evidence.StringIndented(indent+"  "),
		indent, b.LastCommit.StringIndented(indent+"  "),
		indent, b.Hash())
}

// StringShort returns a shortened string representation of the block
func (b *Block) StringShort() string {
	if b == nil {
		return "nil-Block"
	}
	return fmt.Sprintf("Block#%v", b.Hash())
}

//-----------------------------------------------------------
// These methods are for Protobuf Compatibility

// Marshal returns the amino encoding.
func (b *Block) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryBare(b)
}

// MarshalTo calls Marshal and copies to the given buffer.
func (b *Block) MarshalTo(data []byte) (int, error) {
	bs, err := b.Marshal()
	if err != nil {
		return -1, err
	}
	return copy(data, bs), nil
}

// Unmarshal deserializes from amino encoded form.
func (b *Block) Unmarshal(bs []byte) error {
	return cdc.UnmarshalBinaryBare(bs, b)
}

//-----------------------------------------------------------------------------

// MaxDataBytes returns the maximum size of block's data.
//
// XXX: Panics on negative result.
func MaxDataBytes(maxBytes int64, valsCount, evidenceCount int) int64 {
	maxDataBytes := maxBytes -
		MaxAminoOverheadForBlock -
		MaxHeaderBytes -
		int64(valsCount)*MaxVoteBytes -
		int64(evidenceCount)*MaxEvidenceBytes

	if maxDataBytes < 0 {
		panic(fmt.Sprintf(
			"Negative MaxDataBytes. Block.MaxBytes=%d is too small to accommodate header&lastCommit&evidence=%d",
			maxBytes,
			-(maxDataBytes - maxBytes),
		))
	}

	return maxDataBytes

}

// MaxDataBytesUnknownEvidence returns the maximum size of block's data when
// evidence count is unknown. MaxEvidencePerBlock will be used for the size
// of evidence.
//
// XXX: Panics on negative result.
func MaxDataBytesUnknownEvidence(maxBytes int64, valsCount int) int64 {
	_, maxEvidenceBytes := MaxEvidencePerBlock(maxBytes)
	maxDataBytes := maxBytes -
		MaxAminoOverheadForBlock -
		MaxHeaderBytes -
		int64(valsCount)*MaxVoteBytes -
		maxEvidenceBytes

	if maxDataBytes < 0 {
		panic(fmt.Sprintf(
			"Negative MaxDataBytesUnknownEvidence. Block.MaxBytes=%d is too small to accommodate header&lastCommit&evidence=%d",
			maxBytes,
			-(maxDataBytes - maxBytes),
		))
	}

	return maxDataBytes
}

//-----------------------------------------------------------------------------

// Header defines the structure of a Tendermint block header.
// NOTE: changes to the Header should be duplicated in:
// - header.Hash()
// - abci.Header
// - https://github.com/tendermint/spec/blob/master/spec/blockchain/blockchain.md
type Header struct {
	// basic block info
	Version version.Consensus `json:"version"`
	ChainID string            `json:"chain_id"`
	Height  int64             `json:"height"`
	Time    time.Time         `json:"time"`

	// prev block info
	LastBlockID BlockID `json:"last_block_id"`

	// hashes of block data
	LastCommitHash tmbytes.HexBytes `json:"last_commit_hash"` // commit from validators from the last block
	DataHash       tmbytes.HexBytes `json:"data_hash"`        // transactions

	// hashes from the app output from the prev block
	ValidatorsHash     tmbytes.HexBytes `json:"validators_hash"`      // validators for the current block
	NextValidatorsHash tmbytes.HexBytes `json:"next_validators_hash"` // validators for the next block
	ConsensusHash      tmbytes.HexBytes `json:"consensus_hash"`       // consensus params for current block
	AppHash            tmbytes.HexBytes `json:"app_hash"`             // state after txs from the previous block
	// root hash of all results from the txs from the previous block
	LastResultsHash tmbytes.HexBytes `json:"last_results_hash"`

	// consensus info
	EvidenceHash    tmbytes.HexBytes `json:"evidence_hash"`    // evidence included in the block
	ProposerAddress Address          `json:"proposer_address"` // original proposer of the block
}

func (h Header) AminoSize(cdc *amino.Codec) int {
	var size int

	versionSize := h.Version.AminoSize()
	if versionSize > 0 {
		size += 1 + amino.UvarintSize(uint64(versionSize)) + versionSize
	}

	if h.ChainID != "" {
		size += 1 + amino.UvarintSize(uint64(len(h.ChainID))) + len(h.ChainID)
	}

	if h.Height != 0 {
		size += 1 + amino.UvarintSize(uint64(h.Height))
	}

	timeSize := amino.TimeSize(h.Time)
	if timeSize > 0 {
		size += 1 + amino.UvarintSize(uint64(timeSize)) + timeSize
	}

	blockIDSize := h.LastBlockID.AminoSize(cdc)
	if blockIDSize > 0 {
		size += 1 + amino.UvarintSize(uint64(blockIDSize)) + blockIDSize
	}

	if len(h.LastCommitHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.LastCommitHash)
	}

	if len(h.DataHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.DataHash)
	}

	if len(h.ValidatorsHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.ValidatorsHash)
	}

	if len(h.NextValidatorsHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.NextValidatorsHash)
	}

	if len(h.ConsensusHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.ConsensusHash)
	}

	if len(h.AppHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.AppHash)
	}

	if len(h.LastResultsHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.LastResultsHash)
	}

	if len(h.EvidenceHash) != 0 {
		size += 1 + amino.ByteSliceSize(h.EvidenceHash)
	}

	if len(h.ProposerAddress) != 0 {
		size += 1 + amino.ByteSliceSize(h.ProposerAddress)
	}

	return size
}

func (h *Header) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte
	var timeUpdated = false

	for {
		data = data[dataLen:]

		if len(data) == 0 {
			break
		}

		pos, aminoType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if aminoType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}

			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("not enough data for field, need %d, have %d", dataLen, len(data))
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			err = h.Version.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		case 2:
			h.ChainID = string(subData)
		case 3:
			uvint, n, err := amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			h.Height = int64(uvint)
			dataLen = uint64(n)
		case 4:
			h.Time, _, err = amino.DecodeTime(subData)
			if err != nil {
				return err
			}
			timeUpdated = true
		case 5:
			err = h.LastBlockID.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		case 6:
			h.LastCommitHash = make([]byte, len(subData))
			copy(h.LastCommitHash, subData)
		case 7:
			h.DataHash = make([]byte, len(subData))
			copy(h.DataHash, subData)
		case 8:
			h.ValidatorsHash = make([]byte, len(subData))
			copy(h.ValidatorsHash, subData)
		case 9:
			h.NextValidatorsHash = make([]byte, len(subData))
			copy(h.NextValidatorsHash, subData)
		case 10:
			h.ConsensusHash = make([]byte, len(subData))
			copy(h.ConsensusHash, subData)
		case 11:
			h.AppHash = make([]byte, len(subData))
			copy(h.AppHash, subData)
		case 12:
			h.LastResultsHash = make([]byte, len(subData))
			copy(h.LastResultsHash, subData)
		case 13:
			h.EvidenceHash = make([]byte, len(subData))
			copy(h.EvidenceHash, subData)
		case 14:
			h.ProposerAddress = make([]byte, len(subData))
			copy(h.ProposerAddress, subData)
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	if !timeUpdated {
		h.Time = amino.ZeroTime
	}
	return nil
}

// Populate the Header with state-derived data.
// Call this after MakeBlock to complete the Header.
func (h *Header) Populate(
	version version.Consensus, chainID string,
	timestamp time.Time, lastBlockID BlockID,
	valHash, nextValHash []byte,
	consensusHash, appHash, lastResultsHash []byte,
	proposerAddress Address,
) {
	h.Version = version
	h.ChainID = chainID
	h.Time = timestamp
	h.LastBlockID = lastBlockID
	h.ValidatorsHash = valHash
	h.NextValidatorsHash = nextValHash
	h.ConsensusHash = consensusHash
	h.AppHash = appHash
	h.LastResultsHash = lastResultsHash
	h.ProposerAddress = proposerAddress
}

// ValidateBasic performs stateless validation on a Header returning an error
// if any validation fails.
//
// NOTE: Timestamp validation is subtle and handled elsewhere.
func (h Header) ValidateBasic() error {
	if len(h.ChainID) > MaxChainIDLen {
		return fmt.Errorf("chainID is too long; got: %d, max: %d", len(h.ChainID), MaxChainIDLen)
	}

	if h.Height < 0 {
		return errors.New("negative Height")
	} else if h.Height == 0 {
		return errors.New("zero Height")
	}

	if err := h.LastBlockID.ValidateBasic(); err != nil {
		return fmt.Errorf("wrong LastBlockID: %w", err)
	}

	if err := ValidateHash(h.LastCommitHash); err != nil {
		return fmt.Errorf("wrong LastCommitHash: %v", err)
	}

	if err := ValidateHash(h.DataHash); err != nil {
		return fmt.Errorf("wrong DataHash: %v", err)
	}

	if err := ValidateHash(h.EvidenceHash); err != nil {
		return fmt.Errorf("wrong EvidenceHash: %v", err)
	}

	if len(h.ProposerAddress) != crypto.AddressSize {
		return fmt.Errorf(
			"invalid ProposerAddress length; got: %d, expected: %d",
			len(h.ProposerAddress), crypto.AddressSize,
		)
	}

	// Basic validation of hashes related to application data.
	// Will validate fully against state in state#ValidateBlock.
	if err := ValidateHash(h.ValidatorsHash); err != nil {
		return fmt.Errorf("wrong ValidatorsHash: %v", err)
	}
	if err := ValidateHash(h.NextValidatorsHash); err != nil {
		return fmt.Errorf("wrong NextValidatorsHash: %v", err)
	}
	if err := ValidateHash(h.ConsensusHash); err != nil {
		return fmt.Errorf("wrong ConsensusHash: %v", err)
	}
	// NOTE: AppHash is arbitrary length
	if err := ValidateHash(h.LastResultsHash); err != nil {
		return fmt.Errorf("wrong LastResultsHash: %v", err)
	}

	return nil
}

// Hash returns the hash of the header.
// It computes a Merkle tree from the header fields
// ordered as they appear in the Header.
// Returns nil if ValidatorHash is missing,
// since a Header is not valid unless there is
// a ValidatorsHash (corresponding to the validator set).

func (h *Header) Hash() tmbytes.HexBytes {
	if h == nil || len(h.ValidatorsHash) == 0 {
		return nil
	}
	return h.IBCHash()
}
func (h *Header) originHash() tmbytes.HexBytes {
	return merkle.SimpleHashFromByteSlices([][]byte{
		cdcEncode(h.Version),
		cdcEncode(h.ChainID),
		cdcEncode(h.Height),
		cdcEncode(h.Time),
		cdcEncode(h.LastBlockID),
		cdcEncode(h.LastCommitHash),
		cdcEncode(h.DataHash),
		cdcEncode(h.ValidatorsHash),
		cdcEncode(h.NextValidatorsHash),
		cdcEncode(h.ConsensusHash),
		cdcEncode(h.AppHash),
		cdcEncode(h.LastResultsHash),
		cdcEncode(h.EvidenceHash),
		cdcEncode(h.ProposerAddress),
	})
}

func (h *Header) IBCHash() tmbytes.HexBytes {
	if h == nil || len(h.ValidatorsHash) == 0 {
		return nil
	}
	hbz, err := h.Version.Marshal()
	if err != nil {
		return nil
	}
	pbt, err := gogotypes.StdTimeMarshal(h.Time)
	if err != nil {
		return nil
	}

	pbbi := h.LastBlockID.ToIBCProto()
	bzbi, err := pbbi.Marshal()
	if err != nil {
		return nil
	}
	ret := merkle.HashFromByteSlices([][]byte{
		hbz,
		ibccdcEncode(h.ChainID),
		ibccdcEncode(h.Height),
		pbt,
		bzbi,
		ibccdcEncode(h.LastCommitHash),
		ibccdcEncode(h.DataHash),
		ibccdcEncode(h.ValidatorsHash),
		ibccdcEncode(h.NextValidatorsHash),
		ibccdcEncode(h.ConsensusHash),
		ibccdcEncode(h.AppHash),
		ibccdcEncode(h.LastResultsHash),
		ibccdcEncode(h.EvidenceHash),
		ibccdcEncode(h.ProposerAddress),
	})
	return ret
}

// StringIndented returns a string representation of the header
func (h *Header) StringIndented(indent string) string {
	if h == nil {
		return "nil-Header"
	}
	return fmt.Sprintf(`Header{
%s  Version:        %v
%s  ChainID:        %v
%s  Height:         %v
%s  Time:           %v
%s  LastBlockID:    %v
%s  LastCommit:     %v
%s  Data:           %v
%s  Validators:     %v
%s  NextValidators: %v
%s  App:            %v
%s  Consensus:      %v
%s  Results:        %v
%s  Evidence:       %v
%s  Proposer:       %v
%s}#%v`,
		indent, h.Version,
		indent, h.ChainID,
		indent, h.Height,
		indent, h.Time,
		indent, h.LastBlockID,
		indent, h.LastCommitHash,
		indent, h.DataHash,
		indent, h.ValidatorsHash,
		indent, h.NextValidatorsHash,
		indent, h.AppHash,
		indent, h.ConsensusHash,
		indent, h.LastResultsHash,
		indent, h.EvidenceHash,
		indent, h.ProposerAddress,
		indent, h.Hash())
}

// ToProto converts Header to protobuf
func (h *Header) ToProto() *tmproto.Header {
	if h == nil {
		return nil
	}
	return &tmproto.Header{
		Version:            tmversion.Consensus{Block: h.Version.Block.Uint64(), App: h.Version.App.Uint64()},
		ChainID:            h.ChainID,
		Height:             h.Height,
		Time:               h.Time,
		LastBlockID:        h.LastBlockID.ToProto(),
		ValidatorsHash:     h.ValidatorsHash,
		NextValidatorsHash: h.NextValidatorsHash,
		ConsensusHash:      h.ConsensusHash,
		AppHash:            h.AppHash,
		DataHash:           h.DataHash,
		EvidenceHash:       h.EvidenceHash,
		LastResultsHash:    h.LastResultsHash,
		LastCommitHash:     h.LastCommitHash,
		ProposerAddress:    h.ProposerAddress,
	}
}

// FromProto sets a protobuf Header to the given pointer.
// It returns an error if the header is invalid.
func HeaderFromProto(ph *tmproto.Header) (Header, error) {
	if ph == nil {
		return Header{}, errors.New("nil Header")
	}

	h := new(Header)

	bi, err := BlockIDFromProto(&ph.LastBlockID)
	if err != nil {
		return Header{}, err
	}

	h.Version = version.Consensus{Block: version.Protocol(ph.Version.Block), App: version.Protocol(ph.Version.App)}
	h.ChainID = ph.ChainID
	h.Height = ph.Height
	h.Time = ph.Time
	h.Height = ph.Height
	h.LastBlockID = *bi
	h.ValidatorsHash = ph.ValidatorsHash
	h.NextValidatorsHash = ph.NextValidatorsHash
	h.ConsensusHash = ph.ConsensusHash
	h.AppHash = ph.AppHash
	h.DataHash = ph.DataHash
	h.EvidenceHash = ph.EvidenceHash
	h.LastResultsHash = ph.LastResultsHash
	h.LastCommitHash = ph.LastCommitHash
	h.ProposerAddress = ph.ProposerAddress

	return *h, h.ValidateBasic()
}

//-------------------------------------

// BlockIDFlag indicates which BlockID the signature is for.
type BlockIDFlag byte

const (
	// BlockIDFlagAbsent - no vote was received from a validator.
	BlockIDFlagAbsent BlockIDFlag = iota + 1
	// BlockIDFlagCommit - voted for the Commit.BlockID.
	BlockIDFlagCommit
	// BlockIDFlagNil - voted for nil.
	BlockIDFlagNil
)

// CommitSig is a part of the Vote included in a Commit.
type CommitSig struct {
	BlockIDFlag      BlockIDFlag `json:"block_id_flag"`
	ValidatorAddress Address     `json:"validator_address"`
	Timestamp        time.Time   `json:"timestamp"`
	Signature        []byte      `json:"signature"`
}

func (cs CommitSig) AminoSize(_ *amino.Codec) int {
	var size = 0

	if cs.BlockIDFlag != 0 {
		size += 1 + amino.UvarintSize(uint64(cs.BlockIDFlag))
	}

	if len(cs.ValidatorAddress) != 0 {
		size += 1 + amino.ByteSliceSize(cs.ValidatorAddress)
	}

	timestampSize := amino.TimeSize(cs.Timestamp)
	if timestampSize > 0 {
		size += 1 + amino.UvarintSize(uint64(timestampSize)) + timestampSize
	}

	if len(cs.Signature) != 0 {
		size += 1 + amino.ByteSliceSize(cs.Signature)
	}

	return size
}

func (cs *CommitSig) UnmarshalFromAmino(_ *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte
	var timestampUpdated bool

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
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("invalid data len")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			u64, n, err := amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			cs.BlockIDFlag = BlockIDFlag(u64)
			dataLen = uint64(n)
		case 2:
			cs.ValidatorAddress = make([]byte, len(subData))
			copy(cs.ValidatorAddress, subData)
		case 3:
			cs.Timestamp, _, err = amino.DecodeTime(subData)
			if err != nil {
				return err
			}
			timestampUpdated = true
		case 4:
			cs.Signature = make([]byte, len(subData))
			copy(cs.Signature, subData)
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	if !timestampUpdated {
		cs.Timestamp = amino.ZeroTime
	}
	return nil
}

// NewCommitSigForBlock returns new CommitSig with BlockIDFlagCommit.
func NewCommitSigForBlock(signature []byte, valAddr Address, ts time.Time) CommitSig {
	return CommitSig{
		BlockIDFlag:      BlockIDFlagCommit,
		ValidatorAddress: valAddr,
		Timestamp:        ts,
		Signature:        signature,
	}
}

// ForBlock returns true if CommitSig is for the block.
func (cs CommitSig) ForBlock() bool {
	return cs.BlockIDFlag == BlockIDFlagCommit
}

// NewCommitSigAbsent returns new CommitSig with BlockIDFlagAbsent. Other
// fields are all empty.
func NewCommitSigAbsent() CommitSig {
	return CommitSig{
		BlockIDFlag: BlockIDFlagAbsent,
	}
}

// Absent returns true if CommitSig is absent.
func (cs CommitSig) Absent() bool {
	return cs.BlockIDFlag == BlockIDFlagAbsent
}

func (cs CommitSig) String() string {
	return fmt.Sprintf("CommitSig{%X by %X on %v @ %s}",
		tmbytes.Fingerprint(cs.Signature),
		tmbytes.Fingerprint(cs.ValidatorAddress),
		cs.BlockIDFlag,
		CanonicalTime(cs.Timestamp))
}

// BlockID returns the Commit's BlockID if CommitSig indicates signing,
// otherwise - empty BlockID.
func (cs CommitSig) BlockID(commitBlockID BlockID) BlockID {
	var blockID BlockID
	switch cs.BlockIDFlag {
	case BlockIDFlagAbsent:
		blockID = BlockID{}
	case BlockIDFlagCommit:
		blockID = commitBlockID
	case BlockIDFlagNil:
		blockID = BlockID{}
	default:
		panic(fmt.Sprintf("Unknown BlockIDFlag: %v", cs.BlockIDFlag))
	}
	return blockID
}

// ValidateBasic performs basic validation.
func (cs CommitSig) ValidateBasic() error {
	switch cs.BlockIDFlag {
	case BlockIDFlagAbsent:
	case BlockIDFlagCommit:
	case BlockIDFlagNil:
	default:
		return fmt.Errorf("unknown BlockIDFlag: %v", cs.BlockIDFlag)
	}

	switch cs.BlockIDFlag {
	case BlockIDFlagAbsent:
		if len(cs.ValidatorAddress) != 0 {
			return errors.New("validator address is present")
		}
		if !cs.Timestamp.IsZero() {
			return errors.New("time is present")
		}
		if len(cs.Signature) != 0 {
			return errors.New("signature is present")
		}
	default:
		if len(cs.ValidatorAddress) != crypto.AddressSize {
			return fmt.Errorf("expected ValidatorAddress size to be %d bytes, got %d bytes",
				crypto.AddressSize,
				len(cs.ValidatorAddress),
			)
		}
		// NOTE: Timestamp validation is subtle and handled elsewhere.
		if len(cs.Signature) == 0 {
			return errors.New("signature is missing")
		}
		if len(cs.Signature) > MaxSignatureSize {
			return fmt.Errorf("signature is too big (max: %d)", MaxSignatureSize)
		}
	}

	return nil
}

// ToProto converts CommitSig to protobuf
func (cs *CommitSig) ToProto() *tmproto.CommitSig {
	if cs == nil {
		return nil
	}

	return &tmproto.CommitSig{
		BlockIdFlag:      tmproto.BlockIDFlag(cs.BlockIDFlag),
		ValidatorAddress: cs.ValidatorAddress,
		Timestamp:        cs.Timestamp,
		Signature:        cs.Signature,
	}
}

// FromProto sets a protobuf CommitSig to the given pointer.
// It returns an error if the CommitSig is invalid.
func (cs *CommitSig) FromProto(csp tmproto.CommitSig) error {

	cs.BlockIDFlag = BlockIDFlag(csp.BlockIdFlag)
	cs.ValidatorAddress = csp.ValidatorAddress
	cs.Timestamp = csp.Timestamp
	cs.Signature = csp.Signature

	return cs.ValidateBasic()
}

//-------------------------------------

// Commit contains the evidence that a block was committed by a set of validators.
// NOTE: Commit is empty for height 1, but never nil.
type Commit struct {
	// NOTE: The signatures are in order of address to preserve the bonded
	// ValidatorSet order.
	// Any peer with a block can gossip signatures by index with a peer without
	// recalculating the active ValidatorSet.
	Height     int64       `json:"height"`
	Round      int         `json:"round"`
	BlockID    BlockID     `json:"block_id"`
	Signatures []CommitSig `json:"signatures"`

	// Memoized in first call to corresponding method.
	// NOTE: can't memoize in constructor because constructor isn't used for
	// unmarshaling.
	hash     tmbytes.HexBytes
	bitArray *bits.BitArray
}

func (commit Commit) AminoSize(cdc *amino.Codec) int {
	var size int = 0

	if commit.Height != 0 {
		size += 1 + amino.UvarintSize(uint64(commit.Height))
	}

	if commit.Round != 0 {
		size += 1 + amino.UvarintSize(uint64(commit.Round))
	}

	blockIDSize := commit.BlockID.AminoSize(cdc)
	if blockIDSize > 0 {
		size += 1 + amino.UvarintSize(uint64(blockIDSize)) + blockIDSize
	}

	for _, sig := range commit.Signatures {
		sigSize := sig.AminoSize(cdc)
		size += 1 + amino.UvarintSize(uint64(sigSize)) + sigSize
	}

	return size
}

func (commit *Commit) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
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
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("invalid data len")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			u64, n, err := amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			commit.Height = int64(u64)
			dataLen = uint64(n)
		case 2:
			u64, n, err := amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			commit.Round = int(u64)
			dataLen = uint64(n)
		case 3:
			err = commit.BlockID.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		case 4:
			var cs CommitSig
			err = cs.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
			commit.Signatures = append(commit.Signatures, cs)
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

// NewCommit returns a new Commit.
func NewCommit(height int64, round int, blockID BlockID, commitSigs []CommitSig) *Commit {
	return &Commit{
		Height:     height,
		Round:      round,
		BlockID:    blockID,
		Signatures: commitSigs,
	}
}

// CommitToVoteSet constructs a VoteSet from the Commit and validator set.
// Panics if signatures from the commit can't be added to the voteset.
// Inverse of VoteSet.MakeCommit().
func CommitToVoteSet(chainID string, commit *Commit, vals *ValidatorSet) *VoteSet {
	voteSet := NewVoteSet(chainID, commit.Height, commit.Round, PrecommitType, vals)
	for idx, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some precommits can be missing.
		}
		added, err := voteSet.AddVote(commit.GetVote(idx))
		if !added || err != nil {
			panic(fmt.Sprintf("Failed to reconstruct LastCommit: %v", err))
		}
	}
	return voteSet
}

// GetVote converts the CommitSig for the given valIdx to a Vote.
// Returns nil if the precommit at valIdx is nil.
// Panics if valIdx >= commit.Size().
func (commit *Commit) GetVote(valIdx int) *Vote {
	commitSig := commit.Signatures[valIdx]
	return &Vote{
		Type:             PrecommitType,
		Height:           commit.Height,
		Round:            commit.Round,
		BlockID:          commitSig.BlockID(commit.BlockID),
		Timestamp:        commitSig.Timestamp,
		ValidatorAddress: commitSig.ValidatorAddress,
		ValidatorIndex:   valIdx,
		Signature:        commitSig.Signature,
	}
}

// VoteSignBytes constructs the SignBytes for the given CommitSig.
// The only unique part of the SignBytes is the Timestamp - all other fields
// signed over are otherwise the same for all validators.
// Panics if valIdx >= commit.Size().
func (commit *Commit) VoteSignBytes(chainID string, valIdx int) []byte {
	return commit.GetVote(valIdx).SignBytes(chainID)
}

// Type returns the vote type of the commit, which is always VoteTypePrecommit
// Implements VoteSetReader.
func (commit *Commit) Type() byte {
	return byte(PrecommitType)
}

// GetHeight returns height of the commit.
// Implements VoteSetReader.
func (commit *Commit) GetHeight() int64 {
	return commit.Height
}

// GetRound returns height of the commit.
// Implements VoteSetReader.
func (commit *Commit) GetRound() int {
	return commit.Round
}

// Size returns the number of signatures in the commit.
// Implements VoteSetReader.
func (commit *Commit) Size() int {
	if commit == nil {
		return 0
	}
	return len(commit.Signatures)
}

// BitArray returns a BitArray of which validators voted for BlockID or nil in this commit.
// Implements VoteSetReader.
func (commit *Commit) BitArray() *bits.BitArray {
	if commit.bitArray == nil {
		commit.bitArray = bits.NewBitArray(len(commit.Signatures))
		for i, commitSig := range commit.Signatures {
			// TODO: need to check the BlockID otherwise we could be counting conflicts,
			// not just the one with +2/3 !
			commit.bitArray.SetIndex(i, !commitSig.Absent())
		}
	}
	return commit.bitArray
}

// GetByIndex returns the vote corresponding to a given validator index.
// Panics if `index >= commit.Size()`.
// Implements VoteSetReader.
func (commit *Commit) GetByIndex(valIdx int) *Vote {
	return commit.GetVote(valIdx)
}

// IsCommit returns true if there is at least one signature.
// Implements VoteSetReader.
func (commit *Commit) IsCommit() bool {
	return len(commit.Signatures) != 0
}

// ValidateBasic performs basic validation that doesn't involve state data.
// Does not actually check the cryptographic signatures.
func (commit *Commit) ValidateBasic() error {
	if commit.Height < 0 {
		return errors.New("negative Height")
	}
	if commit.Round < 0 {
		return errors.New("negative Round")
	}
	if commit.Height >= 1 {
		if commit.BlockID.IsZero() {
			return errors.New("commit cannot be for nil block")
		}

		if len(commit.Signatures) == 0 {
			return errors.New("no signatures in commit")
		}
		for i, commitSig := range commit.Signatures {
			if err := commitSig.ValidateBasic(); err != nil {
				return fmt.Errorf("wrong CommitSig #%d: %v", i, err)
			}
		}
	}

	return nil
}

// Hash returns the hash of the commit
func (commit *Commit) Hash() tmbytes.HexBytes {
	if commit == nil {
		return nil
	}
	if commit.hash == nil {
		bs := make([][]byte, len(commit.Signatures))
		for i, commitSig := range commit.Signatures {
			bs[i] = cdcEncode(commitSig)
		}
		commit.hash = merkle.SimpleHashFromByteSlices(bs)
	}
	return commit.hash
}

// StringIndented returns a string representation of the commit
func (commit *Commit) StringIndented(indent string) string {
	if commit == nil {
		return "nil-Commit"
	}
	commitSigStrings := make([]string, len(commit.Signatures))
	for i, commitSig := range commit.Signatures {
		commitSigStrings[i] = commitSig.String()
	}
	return fmt.Sprintf(`Commit{
%s  Height:     %d
%s  Round:      %d
%s  BlockID:    %v
%s  Signatures:
%s    %v
%s}#%v`,
		indent, commit.Height,
		indent, commit.Round,
		indent, commit.BlockID,
		indent,
		indent, strings.Join(commitSigStrings, "\n"+indent+"    "),
		indent, commit.hash)
}

// ToProto converts Commit to protobuf
func (commit *Commit) ToProto() *tmproto.Commit {
	if commit == nil {
		return nil
	}

	c := new(tmproto.Commit)
	sigs := make([]tmproto.CommitSig, len(commit.Signatures))
	for i := range commit.Signatures {
		sigs[i] = *commit.Signatures[i].ToProto()
	}
	c.Signatures = sigs

	c.Height = commit.Height
	c.Round = int32(commit.Round)
	c.BlockID = commit.BlockID.ToProto()
	if commit.hash != nil {
		c.Hash = commit.hash
	}
	c.BitArray = commit.bitArray.ToProto()
	return c
}

// FromProto sets a protobuf Commit to the given pointer.
// It returns an error if the commit is invalid.
func CommitFromProto(cp *tmproto.Commit) (*Commit, error) {
	if cp == nil {
		return nil, errors.New("nil Commit")
	}

	var (
		commit   = new(Commit)
		bitArray *bits.BitArray
	)

	bi, err := BlockIDFromProto(&cp.BlockID)
	if err != nil {
		return nil, err
	}

	bitArray.FromProto(cp.BitArray)

	sigs := make([]CommitSig, len(cp.Signatures))
	for i := range cp.Signatures {
		if err := sigs[i].FromProto(cp.Signatures[i]); err != nil {
			return nil, err
		}
	}
	commit.Signatures = sigs

	commit.Height = cp.Height
	commit.Round = int(cp.Round)
	commit.BlockID = *bi
	commit.hash = cp.Hash
	commit.bitArray = bitArray

	return commit, commit.ValidateBasic()
}

//-----------------------------------------------------------------------------

// SignedHeader is a header along with the commits that prove it.
// It is the basis of the lite client.
type SignedHeader struct {
	*Header `json:"header"`

	Commit *Commit `json:"commit"`
}

// ValidateBasic does basic consistency checks and makes sure the header
// and commit are consistent.
// NOTE: This does not actually check the cryptographic signatures.  Make
// sure to use a Verifier to validate the signatures actually provide a
// significantly strong proof for this header's validity.
func (sh SignedHeader) ValidateBasic(chainID string) error {
	return sh.commonValidateBasic(chainID, false)
}

func (sh SignedHeader) commonValidateBasic(chainID string, isIbc bool) error {
	if sh.Header == nil {
		return errors.New("missing header")
	}
	if sh.Commit == nil {
		return errors.New("missing commit")
	}

	if err := sh.Header.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}
	if err := sh.Commit.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid commit: %w", err)
	}

	if sh.ChainID != chainID {
		return fmt.Errorf("header belongs to another chain %q, not %q", sh.ChainID, chainID)
	}

	// Make sure the header is consistent with the commit.
	if sh.Commit.Height != sh.Height {
		return fmt.Errorf("header and commit height mismatch: %d vs %d", sh.Height, sh.Commit.Height)
	}

	var hhash tmbytes.HexBytes
	if isIbc {
		hhash = sh.PureIBCHash()
	} else {
		hhash = sh.Hash()
	}
	if chash := sh.Commit.BlockID.Hash; !bytes.Equal(hhash, chash) {
		return fmt.Errorf("commit signs block %X, header is block %X", chash, hhash)
	}
	return nil
}

func (sh SignedHeader) String() string {
	return sh.StringIndented("")
}

// StringIndented returns a string representation of the SignedHeader.
func (sh SignedHeader) StringIndented(indent string) string {
	return fmt.Sprintf(`SignedHeader{
%s  %v
%s  %v
%s}`,
		indent, sh.Header.StringIndented(indent+"  "),
		indent, sh.Commit.StringIndented(indent+"  "),
		indent)
}

// ToProto converts SignedHeader to protobuf
func (sh *SignedHeader) ToProto() *tmproto.SignedHeader {
	if sh == nil {
		return nil
	}

	psh := new(tmproto.SignedHeader)
	if sh.Header != nil {
		psh.Header = sh.Header.ToProto()
	}
	if sh.Commit != nil {
		psh.Commit = sh.Commit.ToProto()
	}

	return psh
}

// FromProto sets a protobuf SignedHeader to the given pointer.
// It returns an error if the hader or the commit is invalid.
func SignedHeaderFromProto(shp *tmproto.SignedHeader) (*SignedHeader, error) {
	if shp == nil {
		return nil, errors.New("nil SignedHeader")
	}

	sh := new(SignedHeader)

	if shp.Header != nil {
		h, err := HeaderFromProto(shp.Header)
		if err != nil {
			return nil, err
		}
		sh.Header = &h
	}

	if shp.Commit != nil {
		c, err := CommitFromProto(shp.Commit)
		if err != nil {
			return nil, err
		}
		sh.Commit = c
	}

	return sh, nil
}

//-----------------------------------------------------------------------------

// Data contains the set of transactions included in the block
type Data struct {

	// Txs that will be applied by state @ block.Height+1.
	// NOTE: not all txs here are valid.  We're just agreeing on the order first.
	// This means that block.AppHash does not include these txs.
	Txs Txs `json:"txs"`

	txWithMetas TxWithMetas // TxWithMetas is no need participat calculate

	// Volatile
	hash tmbytes.HexBytes
}

func (d Data) AminoSize(_ *amino.Codec) int {
	var size = 0

	for _, tx := range d.Txs {
		size += 1 + amino.ByteSliceSize(tx)
	}

	return size
}

func (d *Data) UnmarshalFromAmino(_ *amino.Codec, data []byte) error {
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
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("invalid data len")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			var tx []byte
			if dataLen > 0 {
				tx = make([]byte, len(subData))
				copy(tx, subData)
			}
			d.Txs = append(d.Txs, tx)
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

// Hash returns the hash of the data
func (data *Data) Hash(height int64) tmbytes.HexBytes {
	if data == nil {
		return (Txs{}).Hash()
	}
	if data.hash == nil {
		if len(data.txWithMetas) > 0 {
			return data.txWithMetas.Hash() // NOTE: leaves of merkle tree are TxIDs
		}
		data.txWithMetas = TxsToTxWithMetas(data.Txs)
		data.hash = data.txWithMetas.Hash() // NOTE: leaves of merkle tree are TxIDs
	}
	return data.hash
}

// StringIndented returns a string representation of the transactions
func (data *Data) StringIndented(indent string) string {
	if data == nil {
		return "nil-Data"
	}
	txStrings := make([]string, tmmath.MinInt(len(data.Txs), 21))
	for i, tx := range data.Txs {
		if i == 20 {
			txStrings[i] = fmt.Sprintf("... (%v total)", len(data.Txs))
			break
		}
		txStrings[i] = fmt.Sprintf("%X (%d bytes)", tx.Hash(), len(tx))
	}
	return fmt.Sprintf(`Data{
%s  %v
%s}#%v`,
		indent, strings.Join(txStrings, "\n"+indent+"  "),
		indent, data.hash)
}

func (data *Data) GetTxWithMetas() TxWithMetas {
	if len(data.Txs) != len(data.txWithMetas) {
		data.txWithMetas = TxsToTxWithMetas(data.Txs)
	}
	return data.txWithMetas
}

//-----------------------------------------------------------------------------

// EvidenceData contains any evidence of malicious wrong-doing by validators
type EvidenceData struct {
	Evidence EvidenceList `json:"evidence"`

	// Volatile
	hash tmbytes.HexBytes
}

func (d EvidenceData) AminoSize(cdc *amino.Codec) int {
	var size = 0

	for _, ev := range d.Evidence {
		if ev != nil {
			var evSize int
			if sizer, ok := ev.(amino.Sizer); ok {
				evSize = 4 + sizer.AminoSize(cdc)
			} else {
				evSize = len(cdc.MustMarshalBinaryBare(ev))
			}
			size += 1 + amino.UvarintSize(uint64(evSize)) + evSize
		} else {
			size += 1 + amino.UvarintSize(0)
		}
	}

	return size
}

func (d *EvidenceData) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
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
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("invalid data len")
			}
			subData = data[:dataLen]
		} else {
			return fmt.Errorf("unexpect pb type %d", pbType)
		}

		switch pos {
		case 1:
			var evi Evidence
			if dataLen == 0 {
				d.Evidence = append(d.Evidence, nil)
			} else {
				eviTmp, err := cdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(subData, &evi)
				if err != nil {
					err = cdc.UnmarshalBinaryBare(subData, &evi)
					if err != nil {
						return err
					} else {
						d.Evidence = append(d.Evidence, evi)
					}
				} else {
					d.Evidence = append(d.Evidence, eviTmp.(Evidence))
				}
			}
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

// Hash returns the hash of the data.
func (data *EvidenceData) Hash() tmbytes.HexBytes {
	if data.hash == nil {
		data.hash = data.Evidence.Hash()
	}
	return data.hash
}

// StringIndented returns a string representation of the evidence.
func (data *EvidenceData) StringIndented(indent string) string {
	if data == nil {
		return "nil-Evidence"
	}
	evStrings := make([]string, tmmath.MinInt(len(data.Evidence), 21))
	for i, ev := range data.Evidence {
		if i == 20 {
			evStrings[i] = fmt.Sprintf("... (%v total)", len(data.Evidence))
			break
		}
		evStrings[i] = fmt.Sprintf("Evidence:%v", ev)
	}
	return fmt.Sprintf(`EvidenceData{
%s  %v
%s}#%v`,
		indent, strings.Join(evStrings, "\n"+indent+"  "),
		indent, data.hash)
}

//--------------------------------------------------------------------------------

// BlockID defines the unique ID of a block as its Hash and its PartSetHeader
type BlockID struct {
	Hash        tmbytes.HexBytes `json:"hash"`
	PartsHeader PartSetHeader    `json:"parts"`
}

func (blockID BlockID) AminoSize(_ *amino.Codec) int {
	var size int
	if len(blockID.Hash) > 0 {
		size += 1 + amino.UvarintSize(uint64(len(blockID.Hash))) + len(blockID.Hash)
	}
	headerSize := blockID.PartsHeader.AminoSize()
	if headerSize > 0 {
		size += 1 + amino.UvarintSize(uint64(headerSize)) + headerSize
	}
	return size
}

func (blockID *BlockID) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]

		if len(data) == 0 {
			break
		}

		pos, aminoType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if aminoType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, err = amino.DecodeUvarint(data)
			if err != nil {
				return err
			}

			data = data[n:]
			if len(data) < int(dataLen) {
				return fmt.Errorf("invalid data len")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			blockID.Hash = make([]byte, len(subData))
			copy(blockID.Hash, subData)
		case 2:
			err = blockID.PartsHeader.UnmarshalFromAmino(cdc, subData)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

// Equals returns true if the BlockID matches the given BlockID
func (blockID BlockID) Equals(other BlockID) bool {
	return bytes.Equal(blockID.Hash, other.Hash) &&
		blockID.PartsHeader.Equals(other.PartsHeader)
}

// Key returns a machine-readable string representation of the BlockID
func (blockID BlockID) Key() string {
	bz, err := cdc.MarshalBinaryBare(blockID.PartsHeader)
	if err != nil {
		panic(err)
	}
	return string(blockID.Hash) + string(bz)
}

// ValidateBasic performs basic validation.
func (blockID BlockID) ValidateBasic() error {
	// Hash can be empty in case of POLBlockID in Proposal.
	if err := ValidateHash(blockID.Hash); err != nil {
		return fmt.Errorf("wrong Hash")
	}
	if err := blockID.PartsHeader.ValidateBasic(); err != nil {
		return fmt.Errorf("wrong PartsHeader: %v", err)
	}
	return nil
}

// IsZero returns true if this is the BlockID of a nil block.
func (blockID BlockID) IsZero() bool {
	return len(blockID.Hash) == 0 &&
		blockID.PartsHeader.IsZero()
}

// IsComplete returns true if this is a valid BlockID of a non-nil block.
func (blockID BlockID) IsComplete() bool {
	return len(blockID.Hash) == tmhash.Size &&
		blockID.PartsHeader.Total > 0 &&
		len(blockID.PartsHeader.Hash) == tmhash.Size
}

// String returns a human readable string representation of the BlockID
func (blockID BlockID) String() string {
	return fmt.Sprintf(`%v:%v`, blockID.Hash, blockID.PartsHeader)
}

// ToProto converts BlockID to protobuf
func (blockID *BlockID) ToProto() tmproto.BlockID {
	if blockID == nil {
		return tmproto.BlockID{}
	}

	return tmproto.BlockID{
		Hash:        blockID.Hash,
		PartsHeader: blockID.PartsHeader.ToProto(),
	}
}

func (blockID *BlockID) ToIBCProto() tmproto.BlockID {
	if blockID == nil {
		return tmproto.BlockID{}
	}
	return tmproto.BlockID{
		Hash:        blockID.Hash,
		PartsHeader: blockID.PartsHeader.ToIBCProto(),
	}
}

// FromProto sets a protobuf BlockID to the given pointer.
// It returns an error if the block id is invalid.
func BlockIDFromProto(bID *tmproto.BlockID) (*BlockID, error) {
	if bID == nil {
		return nil, errors.New("nil BlockID")
	}
	blockID := new(BlockID)
	ph, err := PartSetHeaderFromProto(&bID.PartsHeader)
	if err != nil {
		return nil, err
	}

	blockID.PartsHeader = *ph
	blockID.Hash = bID.Hash

	return blockID, blockID.ValidateBasic()
}
