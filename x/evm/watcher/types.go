package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/status-im/keycard-go/hexutils"
	"github.com/tendermint/go-amino"

	app "github.com/okx/okbchain/app/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth"
	abci "github.com/okx/okbchain/libs/tendermint/abci/types"
	"github.com/okx/okbchain/libs/tendermint/crypto/merkle"
	ctypes "github.com/okx/okbchain/libs/tendermint/rpc/core/types"
	"github.com/okx/okbchain/x/evm/types"
)

var (
	prefixTx           = []byte{0x01}
	prefixBlock        = []byte{0x02}
	prefixReceipt      = []byte{0x03}
	prefixCode         = []byte{0x04}
	prefixBlockInfo    = []byte{0x05}
	prefixLatestHeight = []byte{0x06}
	prefixAccount      = []byte{0x07}
	PrefixState        = []byte{0x08}
	prefixCodeHash     = []byte{0x09}
	prefixParams       = []byte{0x10}
	prefixWhiteList    = []byte{0x11}
	prefixBlackList    = []byte{0x12}
	prefixRpcDb        = []byte{0x13}
	prefixTxResponse   = []byte{0x14}
	prefixStdTxHash    = []byte{0x15}

	KeyLatestHeight = "LatestHeight"

	TransactionSuccess = uint32(1)
	TransactionFailed  = uint32(0)

	keyLatestBlockHeight = append(prefixLatestHeight, KeyLatestHeight...)
)

const (
	TypeOthers    = uint32(1)
	TypeState     = uint32(2)
	TypeDelete    = uint32(3)
	TypeEvmParams = uint32(4)

	EthReceipt  = uint64(0)
	StdResponse = uint64(1)
)

type WatchMessage = sdk.WatchMessage

type Batch struct {
	Key       []byte `json:"key"`
	Value     []byte `json:"value"`
	TypeValue uint32 `json:"type_value"`
}

// MarshalToAmino marshal batch data to amino bytes
func (b *Batch) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	fieldKeysType := [3]byte{1<<3 | 2, 2<<3 | 2, 3 << 3}
	for pos := 1; pos <= 3; pos++ {
		switch pos {
		case 1:
			if len(b.Key) == 0 {
				break
			}
			err = buf.WriteByte(fieldKeysType[pos-1])
			if err != nil {
				return nil, err
			}
			err = amino.EncodeByteSliceToBuffer(&buf, b.Key)
			if err != nil {
				return nil, err
			}

		case 2:
			if len(b.Value) == 0 {
				break
			}
			err = buf.WriteByte(fieldKeysType[pos-1])
			if err != nil {
				return nil, err
			}
			err = amino.EncodeByteSliceToBuffer(&buf, b.Value)
			if err != nil {
				return nil, err
			}
		case 3:
			if b.TypeValue == 0 {
				break
			}
			err := buf.WriteByte(fieldKeysType[pos-1])
			if err != nil {
				return nil, err
			}
			err = amino.EncodeUvarintToBuffer(&buf, uint64(b.TypeValue))
			if err != nil {
				return nil, err
			}

		default:
			panic("unreachable")
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalFromAmino unmarshal amino bytes to this object
func (b *Batch) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]
		if len(data) == 0 {
			break
		}
		// decode field key type and data position
		pos, pbType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		// choose sub-data to parse data
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
			b.Key = make([]byte, len(subData))
			copy(b.Key, subData)

		case 2:
			b.Value = make([]byte, len(subData))
			copy(b.Value, subData)

		case 3:
			tv, n, err := amino.DecodeUvarint(data)
			if err != nil {
				return err
			}
			b.TypeValue = uint32(tv)
			dataLen = uint64(n)

		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

type WatchData struct {
	DirtyAccount  []*sdk.AccAddress `json:"dirty_account"`
	Batches       []*Batch          `json:"batches"`
	DelayEraseKey [][]byte          `json:"delay_erase_key"`
	BloomData     []*types.KV       `json:"bloom_data"`
	DirtyList     [][]byte          `json:"dirty_list"`
}

func (w *WatchData) Size() int {
	return len(w.DirtyAccount) + len(w.Batches) + len(w.DelayEraseKey) + len(w.BloomData) + len(w.DirtyList)
}

// MarshalToAmino marshal to amino bytes
func (w *WatchData) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	fieldKeysType := [5]byte{1<<3 | 2, 2<<3 | 2, 3<<3 | 2, 4<<3 | 2, 5<<3 | 2}
	for pos := 1; pos <= 5; pos++ {
		switch pos {
		case 1:
			if len(w.DirtyAccount) == 0 {
				break
			}
			for i := 0; i < len(w.DirtyAccount); i++ {
				err := buf.WriteByte(fieldKeysType[pos-1])
				if err != nil {
					return nil, err
				}
				var data []byte
				if w.DirtyAccount[i] != nil {
					data = w.DirtyAccount[i].Bytes()
				}
				err = amino.EncodeByteSliceToBuffer(&buf, data)
				if err != nil {
					return nil, err
				}

			}
		case 2:
			if len(w.Batches) == 0 {
				break
			}
			for i := 0; i < len(w.Batches); i++ {
				err = buf.WriteByte(fieldKeysType[pos-1])
				if err != nil {
					return nil, err
				}

				var data []byte
				if w.Batches[i] != nil {
					data, err = w.Batches[i].MarshalToAmino(cdc)
					if err != nil {
						return nil, err
					}
				}
				err = amino.EncodeByteSliceToBuffer(&buf, data)
				if err != nil {
					return nil, err
				}
			}
		case 3:
			if len(w.DelayEraseKey) == 0 {
				break
			}
			// encode a slice one by one
			for i := 0; i < len(w.DelayEraseKey); i++ {
				err = buf.WriteByte(fieldKeysType[pos-1])
				if err != nil {
					return nil, err
				}
				err = amino.EncodeByteSliceToBuffer(&buf, w.DelayEraseKey[i])
				if err != nil {
					return nil, err
				}
			}
		case 4:
			if len(w.BloomData) == 0 {
				break
			}
			for i := 0; i < len(w.BloomData); i++ {
				err = buf.WriteByte(fieldKeysType[pos-1])
				if err != nil {
					return nil, err
				}
				var data []byte
				if w.BloomData[i] != nil {
					data, err = w.BloomData[i].MarshalToAmino(cdc)
					if err != nil {
						return nil, err
					}
				}
				err = amino.EncodeByteSliceToBuffer(&buf, data)
				if err != nil {
					return nil, err
				}
			}
		case 5:
			if len(w.DirtyList) == 0 {
				break
			}
			// encode a slice one by one
			for i := 0; i < len(w.DirtyList); i++ {
				err = buf.WriteByte(fieldKeysType[pos-1])
				if err != nil {
					return nil, err
				}
				err = amino.EncodeByteSliceToBuffer(&buf, w.DirtyList[i])
				if err != nil {
					return nil, err
				}
			}
		default:
			panic("unreachable")
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalFromAmino unmarshal from amino bytes
func (w *WatchData) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
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
			// copy subData to new memory and use it as sdk.AccAddress type
			var acc *sdk.AccAddress = nil
			if len(subData) != 0 {
				accAddr := make([]byte, len(subData))
				copy(accAddr, subData)
				accByte := sdk.AccAddress(accAddr)
				acc = &accByte
			}
			w.DirtyAccount = append(w.DirtyAccount, acc)

		case 2:
			var bat *Batch = nil
			if len(subData) != 0 {
				bat = &Batch{}
				err := bat.UnmarshalFromAmino(cdc, subData)
				if err != nil {
					return err
				}
			}
			w.Batches = append(w.Batches, bat)

		case 3:
			var delayEraseKey []byte
			if len(subData) != 0 {
				delayEraseKey = make([]byte, len(subData))
				copy(delayEraseKey, subData)
			}
			w.DelayEraseKey = append(w.DelayEraseKey, delayEraseKey)

		case 4:
			var kv *types.KV = nil
			if len(subData) != 0 {
				kv = &types.KV{}
				err := kv.UnmarshalFromAmino(nil, subData)
				if err != nil {
					return err
				}
			}
			w.BloomData = append(w.BloomData, kv)

		case 5:
			var dirtyList []byte
			if len(subData) != 0 {
				dirtyList = make([]byte, len(subData))
				copy(dirtyList, subData)
			}
			w.DirtyList = append(w.DirtyList, dirtyList)

		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

type MsgCode struct {
	Key  []byte
	Code string
}

func (m MsgCode) GetType() uint32 {
	return TypeOthers
}

type CodeInfo struct {
	Height uint64 `json:"height"`
	Code   string `json:"code"`
}

func NewMsgCode(contractAddr common.Address, code []byte, height uint64) *MsgCode {
	codeInfo := CodeInfo{
		Height: height,
		Code:   hexutils.BytesToHex(code),
	}
	jsCode, e := json.Marshal(codeInfo)
	if e != nil {
		return nil
	}
	return &MsgCode{
		Key:  contractAddr.Bytes(),
		Code: string(jsCode),
	}
}

func (m MsgCode) GetKey() []byte {
	return append(prefixCode, m.Key...)
}

func (m MsgCode) GetValue() string {
	return m.Code
}

type MsgCodeByHash struct {
	Key  []byte
	Code string
}

func (m MsgCodeByHash) GetType() uint32 {
	return TypeOthers
}

func NewMsgCodeByHash(hash []byte, code []byte) *MsgCodeByHash {
	return &MsgCodeByHash{
		Key:  hash,
		Code: string(code),
	}
}

func (m MsgCodeByHash) GetKey() []byte {
	return append(prefixCodeHash, m.Key...)
}

func (m MsgCodeByHash) GetValue() string {
	return m.Code
}

type MsgTransactionReceipt struct {
	*TransactionReceipt
	txHash []byte
}

func (m MsgTransactionReceipt) GetType() uint32 {
	return TypeOthers
}

// type WrappedResponseWithCodec
type WrappedResponseWithCodec struct {
	Response sdk.TxResponse
	Codec    *codec.Codec `json:"-"`
}

type TransactionResult struct {
	TxType   hexutil.Uint64            `json:"type"`
	EthTx    *Transaction              `json:"ethTx"`
	EthTxLog string                    `json:"ethTxLog"`
	Receipt  *TransactionReceipt       `json:"receipt"`
	Response *WrappedResponseWithCodec `json:"response"`
}

func (wr *WrappedResponseWithCodec) MarshalJSON() ([]byte, error) {
	if wr.Codec != nil {
		return wr.Codec.MarshalJSON(wr.Response)
	}

	return json.Marshal(wr.Response)
}

type TransactionReceipt struct {
	Status                hexutil.Uint64  `json:"status"`
	CumulativeGasUsed     hexutil.Uint64  `json:"cumulativeGasUsed"`
	LogsBloom             ethtypes.Bloom  `json:"logsBloom"`
	Logs                  []*ethtypes.Log `json:"logs"`
	originTransactionHash common.Hash
	TransactionHash       string          `json:"transactionHash"`
	ContractAddress       *common.Address `json:"contractAddress"`
	GasUsed               hexutil.Uint64  `json:"gasUsed"`
	originBlockHash       common.Hash
	BlockHash             string         `json:"blockHash"`
	BlockNumber           hexutil.Uint64 `json:"blockNumber"`
	TransactionIndex      hexutil.Uint64 `json:"transactionIndex"`
	tx                    *types.MsgEthereumTx
	From                  string          `json:"from"`
	To                    *common.Address `json:"to"`
}

func (tr *TransactionReceipt) GetValue() string {
	tr.fillInfoForMarshal()
	protoReceipt := receiptToProto(tr)
	buf, err := proto.Marshal(protoReceipt)
	if err != nil {
		panic("cant happen")
	}

	return string(buf)
}

// must call this func before marshaling
func (tr *TransactionReceipt) fillInfoForMarshal() {
	tr.TransactionHash = tr.GetHash()
	tr.BlockHash = tr.GetBlockHash()
	tr.From = tr.GetFrom()
	tr.To = tr.tx.To()
	//contract address will be set to 0x0000000000000000000000000000000000000000 if contract deploy failed
	if tr.ContractAddress != nil && types.EthAddressStringer(*tr.ContractAddress).String() == "0x0000000000000000000000000000000000000000" {
		//set to nil to keep sync with ethereum rpc
		tr.ContractAddress = nil
	}
}

func (tr *TransactionReceipt) GetHash() string {
	return types.EthHashStringer(tr.originTransactionHash).String()
}

func (tr *TransactionReceipt) GetBlockHash() string {
	return types.EthHashStringer(tr.originBlockHash).String()
}

func (tr *TransactionReceipt) GetFrom() string {
	return types.EthAddressStringer(common.BytesToAddress(tr.tx.AccountAddress().Bytes())).String()
}

func (tr *TransactionReceipt) GetTo() *common.Address {
	return tr.tx.To()
}

func newTransactionReceipt(status uint32, tx *types.MsgEthereumTx, txHash, blockHash common.Hash, txIndex, height uint64, data *types.ResultData, cumulativeGas, GasUsed uint64) *TransactionReceipt {
	return &TransactionReceipt{
		Status:                hexutil.Uint64(status),
		CumulativeGasUsed:     hexutil.Uint64(cumulativeGas),
		LogsBloom:             data.Bloom,
		Logs:                  data.Logs,
		originTransactionHash: txHash,
		ContractAddress:       &data.ContractAddress,
		GasUsed:               hexutil.Uint64(GasUsed),
		originBlockHash:       blockHash,
		BlockNumber:           hexutil.Uint64(height),
		TransactionIndex:      hexutil.Uint64(txIndex),
		tx:                    tx,
	}
}

func NewTransactionReceiptResponse(status uint32, tx *types.MsgEthereumTx, txHash, blockHash common.Hash, txIndex, height uint64,
	data *types.ResultData, cumulativeGas, gasUsed uint64) *TransactionReceipt {
	receipt := newTransactionReceipt(status, tx, txHash, blockHash, txIndex, height, data, cumulativeGas, gasUsed)
	//
	receipt.fillInfoForMarshal()
	return receipt
}

func NewMsgTransactionReceipt(tr *TransactionReceipt, txHash common.Hash) *MsgTransactionReceipt {
	return &MsgTransactionReceipt{txHash: txHash.Bytes(), TransactionReceipt: tr}
}

func (m MsgTransactionReceipt) GetKey() []byte {
	return append(prefixReceipt, m.txHash...)
}

type MsgBlock struct {
	blockHash []byte
	block     string
}

func (m MsgBlock) GetType() uint32 {
	return TypeOthers
}

// Transaction represents a transaction returned to RPC clients.
type Transaction struct {
	BlockHash         *common.Hash    `json:"blockHash"`
	BlockNumber       *hexutil.Big    `json:"blockNumber"`
	From              common.Address  `json:"from"`
	Gas               hexutil.Uint64  `json:"gas"`
	GasPrice          *hexutil.Big    `json:"gasPrice"`
	Hash              common.Hash     `json:"hash"`
	Input             hexutil.Bytes   `json:"input"`
	Nonce             hexutil.Uint64  `json:"nonce"`
	To                *common.Address `json:"to"`
	TransactionIndex  *hexutil.Uint64 `json:"transactionIndex"`
	Value             *hexutil.Big    `json:"value"`
	V                 *hexutil.Big    `json:"v"`
	R                 *hexutil.Big    `json:"r"`
	S                 *hexutil.Big    `json:"s"`
	tx                *types.MsgEthereumTx
	originBlockHash   *common.Hash
	originBlockNumber uint64
	originIndex       uint64
}

func (tr *Transaction) GetValue() string {
	// Verify signature and retrieve sender address
	err := tr.tx.VerifySig(tr.tx.ChainID(), int64(tr.originBlockNumber))
	if err != nil {
		return ""
	}

	tr.From = common.HexToAddress(tr.tx.GetFrom())
	tr.Gas = hexutil.Uint64(tr.tx.Data.GasLimit)
	tr.GasPrice = (*hexutil.Big)(tr.tx.Data.Price)
	tr.Input = tr.tx.Data.Payload
	tr.Nonce = hexutil.Uint64(tr.tx.Data.AccountNonce)
	tr.To = tr.tx.To()
	tr.Value = (*hexutil.Big)(tr.tx.Data.Amount)
	tr.V = (*hexutil.Big)(tr.tx.Data.V)
	tr.R = (*hexutil.Big)(tr.tx.Data.R)
	tr.S = (*hexutil.Big)(tr.tx.Data.S)

	if *tr.originBlockHash != (common.Hash{}) {
		tr.BlockHash = tr.originBlockHash
		tr.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(tr.originBlockNumber))
		tr.TransactionIndex = (*hexutil.Uint64)(&tr.originIndex)
	}
	protoTransaction := transactionToProto(tr)
	buf, err := proto.Marshal(protoTransaction)
	if err != nil {
		panic("cant happen")
	}

	return string(buf)
}

func NewBlock(height uint64, blockBloom ethtypes.Bloom, header abci.Header, gasLimit uint64,
	gasUsed *big.Int, txs interface{}) (types.Block, common.Hash) {
	timestamp := header.Time.Unix()
	if timestamp < 0 {
		timestamp = time.Now().Unix()
	}

	transactionsRoot := ethtypes.EmptyRootHash
	if len(txs.([]common.Hash)) > 0 {
		txsHash := txs.([]common.Hash)
		txBzs := make([][]byte, len(txsHash))
		for i := 0; i < len(txsHash); i++ {
			txBzs[i] = txsHash[i].Bytes()
		}
		transactionsRoot = common.BytesToHash(merkle.SimpleHashFromByteSlices(txBzs))
	}

	block := types.Block{
		Number:           hexutil.Uint64(height),
		ParentHash:       common.BytesToHash(header.LastBlockId.Hash),
		Nonce:            types.BlockNonce{},
		UncleHash:        ethtypes.EmptyUncleHash,
		LogsBloom:        blockBloom,
		TransactionsRoot: transactionsRoot,
		StateRoot:        common.BytesToHash(header.AppHash),
		Miner:            common.BytesToAddress(header.ProposerAddress),
		MixHash:          common.Hash{},
		Difficulty:       0,
		TotalDifficulty:  0,
		ExtraData:        nil,
		Size:             hexutil.Uint64(header.Size()),
		GasLimit:         hexutil.Uint64(gasLimit),
		GasUsed:          (*hexutil.Big)(gasUsed),
		Timestamp:        hexutil.Uint64(timestamp),
		Uncles:           []common.Hash{},
		ReceiptsRoot:     ethtypes.EmptyRootHash,
		Transactions:     txs,
	}
	ethBlockHash := block.EthHash()
	block.Hash = ethBlockHash
	return block, ethBlockHash
}

func NewMsgBlock(b types.Block, blockHash common.Hash) *MsgBlock {
	jsBlock, e := json.Marshal(b)
	if e != nil {
		return nil
	}
	return &MsgBlock{blockHash: blockHash.Bytes(), block: string(jsBlock)}
}

func (m MsgBlock) GetKey() []byte {
	return append(prefixBlock, m.blockHash...)
}

func (m MsgBlock) GetValue() string {
	return m.block
}

type MsgBlockInfo struct {
	height []byte
	hash   string
}

func (b MsgBlockInfo) GetType() uint32 {
	return TypeOthers
}

func NewMsgBlockInfo(height uint64, blockHash common.Hash) *MsgBlockInfo {
	return &MsgBlockInfo{
		height: []byte(strconv.Itoa(int(height))),
		hash:   blockHash.String(),
	}
}

func (b MsgBlockInfo) GetKey() []byte {
	return append(prefixBlockInfo, b.height...)
}

func (b MsgBlockInfo) GetValue() string {
	return b.hash
}

type MsgLatestHeight struct {
	height string
}

func (b MsgLatestHeight) GetType() uint32 {
	return TypeOthers
}

func NewMsgLatestHeight(height uint64) *MsgLatestHeight {
	return &MsgLatestHeight{
		height: strconv.Itoa(int(height)),
	}
}

func (b MsgLatestHeight) GetKey() []byte {
	return append(prefixLatestHeight, KeyLatestHeight...)
}

func (b MsgLatestHeight) GetValue() string {
	return b.height
}

type MsgAccount struct {
	account *app.EthAccount
}

func (msgAccount *MsgAccount) GetType() uint32 {
	return TypeOthers
}

func NewMsgAccount(acc auth.Account) *MsgAccount {
	var msg *MsgAccount
	switch v := acc.(type) {
	case app.EthAccount:
		msg = &MsgAccount{account: &v}
	case *app.EthAccount:
		msg = &MsgAccount{account: v}
	default:
		msg = nil
	}
	return msg
}

func GetMsgAccountKey(addr []byte) []byte {
	return append(prefixAccount, addr...)
}

func (msgAccount *MsgAccount) GetKey() []byte {
	return GetMsgAccountKey(msgAccount.account.Address.Bytes())
}

func (msgAccount *MsgAccount) GetValue() string {
	data, err := EncodeAccount(msgAccount.account)
	if err != nil {
		panic(err)
	}
	return string(data)
}

type DelAccMsg struct {
	addr []byte
}

func NewDelAccMsg(acc auth.Account) *DelAccMsg {
	return &DelAccMsg{
		addr: acc.GetAddress().Bytes(),
	}
}

func (delAcc *DelAccMsg) GetType() uint32 {
	return TypeDelete
}

func (delAcc *DelAccMsg) GetKey() []byte {
	return GetMsgAccountKey(delAcc.addr)
}

func (delAcc *DelAccMsg) GetValue() string {
	return ""
}

type MsgState struct {
	addr  common.Address
	key   []byte
	value []byte
}

func (msgState *MsgState) GetType() uint32 {
	return TypeState
}

func NewMsgState(addr common.Address, key, value []byte) *MsgState {
	return &MsgState{
		addr:  addr,
		key:   key,
		value: value,
	}
}

func GetMsgStateKey(addr common.Address, key []byte) []byte {
	prefix := addr.Bytes()
	compositeKey := make([]byte, len(prefix)+len(key))

	copy(compositeKey, prefix)
	copy(compositeKey[len(prefix):], key)

	return append(PrefixState, compositeKey...)
}

func (msgState *MsgState) GetKey() []byte {
	return GetMsgStateKey(msgState.addr, msgState.key)
}

func (msgState *MsgState) GetValue() string {
	return string(msgState.value)
}

type MsgParams struct {
	types.Params
}

func (msgParams *MsgParams) GetType() uint32 {
	return TypeEvmParams
}

func NewMsgParams(params types.Params) *MsgParams {
	return &MsgParams{
		params,
	}
}

func (msgParams *MsgParams) GetKey() []byte {
	return prefixParams
}

func (msgParams *MsgParams) GetValue() string {
	jsonValue, err := json.Marshal(msgParams)
	if err != nil {
		panic(err)
	}
	return string(jsonValue)
}

type MsgContractBlockedListItem struct {
	addr sdk.AccAddress
}

func (msgItem *MsgContractBlockedListItem) GetType() uint32 {
	return TypeOthers
}

func NewMsgContractBlockedListItem(addr sdk.AccAddress) *MsgContractBlockedListItem {
	return &MsgContractBlockedListItem{
		addr: addr,
	}
}

func (msgItem *MsgContractBlockedListItem) GetKey() []byte {
	return append(prefixBlackList, msgItem.addr.Bytes()...)
}

func (msgItem *MsgContractBlockedListItem) GetValue() string {
	return ""
}

type MsgDelContractBlockedListItem struct {
	addr sdk.AccAddress
}

func (msgItem *MsgDelContractBlockedListItem) GetType() uint32 {
	return TypeDelete
}

func NewMsgDelContractBlockedListItem(addr sdk.AccAddress) *MsgDelContractBlockedListItem {
	return &MsgDelContractBlockedListItem{
		addr: addr,
	}
}

func (msgItem *MsgDelContractBlockedListItem) GetKey() []byte {
	return append(prefixBlackList, msgItem.addr.Bytes()...)
}

func (msgItem *MsgDelContractBlockedListItem) GetValue() string {
	return ""
}

type MsgContractDeploymentWhitelistItem struct {
	addr sdk.AccAddress
}

func (msgItem *MsgContractDeploymentWhitelistItem) GetType() uint32 {
	return TypeOthers
}

func NewMsgContractDeploymentWhitelistItem(addr sdk.AccAddress) *MsgContractDeploymentWhitelistItem {
	return &MsgContractDeploymentWhitelistItem{
		addr: addr,
	}
}

func (msgItem *MsgContractDeploymentWhitelistItem) GetKey() []byte {
	return append(prefixWhiteList, msgItem.addr.Bytes()...)
}

func (msgItem *MsgContractDeploymentWhitelistItem) GetValue() string {
	return ""
}

type MsgDelContractDeploymentWhitelistItem struct {
	addr sdk.AccAddress
}

func (msgItem *MsgDelContractDeploymentWhitelistItem) GetType() uint32 {
	return TypeDelete
}

func NewMsgDelContractDeploymentWhitelistItem(addr sdk.AccAddress) *MsgDelContractDeploymentWhitelistItem {
	return &MsgDelContractDeploymentWhitelistItem{
		addr: addr,
	}
}

func (msgItem *MsgDelContractDeploymentWhitelistItem) GetKey() []byte {
	return append(prefixWhiteList, msgItem.addr.Bytes()...)
}

func (msgItem *MsgDelContractDeploymentWhitelistItem) GetValue() string {
	return ""
}

type MsgContractMethodBlockedListItem struct {
	addr    sdk.AccAddress
	methods []byte
}

func (msgItem *MsgContractMethodBlockedListItem) GetType() uint32 {
	return TypeOthers
}

func NewMsgContractMethodBlockedListItem(addr sdk.AccAddress, methods []byte) *MsgContractMethodBlockedListItem {
	return &MsgContractMethodBlockedListItem{
		addr:    addr,
		methods: methods,
	}
}

func (msgItem *MsgContractMethodBlockedListItem) GetKey() []byte {
	return append(prefixBlackList, msgItem.addr.Bytes()...)
}

func (msgItem *MsgContractMethodBlockedListItem) GetValue() string {
	return string(msgItem.methods)
}

type MsgStdTransactionResponse struct {
	txResponse string
	txHash     []byte
}

func (tr *MsgStdTransactionResponse) GetType() uint32 {
	return TypeOthers
}

func (tr *MsgStdTransactionResponse) GetValue() string {
	return tr.txResponse
}

func (tr *MsgStdTransactionResponse) GetKey() []byte {
	return append(prefixTxResponse, tr.txHash...)
}

type TransactionResponse struct {
	*ctypes.ResultTx
	Timestamp time.Time
}

func NewStdTransactionResponse(tr *ctypes.ResultTx, timestamp time.Time, txHash common.Hash) *MsgStdTransactionResponse {
	tResponse := TransactionResponse{
		ResultTx:  tr,
		Timestamp: timestamp,
	}
	jsResponse, err := json.Marshal(tResponse)

	if err != nil {
		return nil
	}
	return &MsgStdTransactionResponse{txResponse: string(jsResponse), txHash: txHash.Bytes()}
}

type MsgBlockStdTxHash struct {
	blockHash []byte
	stdTxHash string
}

func (m *MsgBlockStdTxHash) GetType() uint32 {
	return TypeOthers
}

func (m *MsgBlockStdTxHash) GetValue() string {
	return m.stdTxHash
}

func (m *MsgBlockStdTxHash) GetKey() []byte {
	return append(prefixStdTxHash, m.blockHash...)
}

func NewMsgBlockStdTxHash(stdTxHash []common.Hash, blockHash common.Hash) *MsgBlockStdTxHash {
	jsonValue, err := json.Marshal(stdTxHash)
	if err != nil {
		panic(err)
	}

	return &MsgBlockStdTxHash{
		stdTxHash: string(jsonValue),
		blockHash: blockHash.Bytes(),
	}
}
