// nolint
package mock

import (
	"bytes"
	"fmt"
	"math/big"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
)

// An sdk.Tx which is its own sdk.Msg.
type kvstoreTx struct {
	sdk.BaseTx

	key   []byte
	value []byte
	bytes []byte
}

var _ sdk.Tx = (*kvstoreTx)(nil)

func NewTx(key, value string) *kvstoreTx {
	bytes := fmt.Sprintf("%s=%s", key, value)
	return &kvstoreTx{
		key:   []byte(key),
		value: []byte(value),
		bytes: []byte(bytes),
	}
}

func (tx *kvstoreTx) Route() string {
	return "kvstore"
}

func (tx *kvstoreTx) Type() string {
	return "kvstore_tx"
}

func (tx *kvstoreTx) GetMsgs() []sdk.Msg {
	return []sdk.Msg{tx}
}

func (tx *kvstoreTx) GetMemo() string {
	return ""
}

func (tx *kvstoreTx) GetSignBytes() []byte {
	return tx.bytes
}

// Should the app be calling this? Or only handlers?
func (tx *kvstoreTx) ValidateBasic() error {
	return nil
}

func (tx *kvstoreTx) GetSigners() []sdk.AccAddress {
	return nil
}

func (tx *kvstoreTx) GetType() sdk.TransactionType {
	return sdk.UnknownType
}

func (tx *kvstoreTx) GetFrom() string {
	return ""
}

func (tx *kvstoreTx) GetSender(_ sdk.Context) string {
	return ""
}

func (tx *kvstoreTx) GetNonce() uint64 {
	return 0
}

func (tx *kvstoreTx) GetGasPrice() *big.Int {
	return big.NewInt(0)
}

func (tx *kvstoreTx) GetTxFnSignatureInfo() ([]byte, int) {
	return nil, 0
}

func (tx *kvstoreTx) GetGas() uint64 {
	return 0
}

// takes raw transaction bytes and decodes them into an sdk.Tx. An sdk.Tx has
// all the signatures and can be used to authenticate.
func decodeTx(txBytes []byte, _ ...int64) (sdk.Tx, error) {
	var tx sdk.Tx

	split := bytes.Split(txBytes, []byte("="))
	if len(split) == 1 {
		k := split[0]
		tx = &kvstoreTx{key: k, value: k, bytes: txBytes}
	} else if len(split) == 2 {
		k, v := split[0], split[1]
		tx = &kvstoreTx{key: k, value: v, bytes: txBytes}
	} else {
		return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "too many '='")
	}

	return tx, nil
}

func decodeTxWithHash(txBytes, txHash []byte, _ ...int64) (sdk.Tx, error) {
	return decodeTx(txBytes)
}
