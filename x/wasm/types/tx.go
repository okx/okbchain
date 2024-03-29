package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	ethcmm "github.com/ethereum/go-ethereum/common"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
)

// RawContractMessage defines a json message that is sent or returned by a wasm contract.
// This type can hold any type of bytes. Until validateBasic is called there should not be
// any assumptions made that the data is valid syntax or semantic.
type RawContractMessage []byte

func (r RawContractMessage) MarshalJSON() ([]byte, error) {
	return json.RawMessage(r).MarshalJSON()
}

func (r *RawContractMessage) UnmarshalJSON(b []byte) error {
	if r == nil {
		return errors.New("unmarshalJSON on nil pointer")
	}
	*r = append((*r)[0:0], b...)
	return nil
}

func (r *RawContractMessage) ValidateBasic() error {
	if r == nil {
		return ErrEmpty
	}
	if !json.Valid(*r) {
		return ErrInvalid
	}
	return nil
}

// Bytes returns raw bytes type
func (r RawContractMessage) Bytes() []byte {
	return r
}

func (msg MsgStoreCode) Route() string {
	return RouterKey
}

func (msg MsgStoreCode) Type() string {
	return "store-code"
}

func (msg MsgStoreCode) ValidateBasic() error {
	if _, err := sdk.WasmAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if err := validateWasmCode(msg.WASMByteCode); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "code bytes %s", err.Error())
	}

	if msg.InstantiatePermission != nil {
		if err := msg.InstantiatePermission.ValidateBasic(); err != nil {
			return sdkerrors.Wrap(err, "instantiate permission")
		}
	}
	return nil
}

func (msg MsgStoreCode) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgStoreCode) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.WasmAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{sdk.WasmToAccAddress(senderAddr)}
}

func (msg MsgStoreCode) FnSignatureInfo() (string, int, error) {
	codeLen := len(msg.WASMByteCode)
	var err error
	if codeLen <= 0 {
		err = fmt.Errorf("wasm byte code length is 0")
	}

	return msg.Type(), codeLen, err
}

func (msg MsgInstantiateContract) Route() string {
	return RouterKey
}

func (msg MsgInstantiateContract) Type() string {
	return "instantiate"
}

func (msg MsgInstantiateContract) ValidateBasic() error {
	if _, err := sdk.WasmAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}

	if msg.CodeID == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "code id is required")
	}

	if err := validateLabel(msg.Label); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "label is required")
	}

	if !msg.Funds.IsValid() {
		return sdkerrors.ErrInvalidCoins
	}

	if len(msg.Admin) != 0 {
		if _, err := sdk.WasmAddressFromBech32(msg.Admin); err != nil {
			return sdkerrors.Wrap(err, "admin")
		}
	}
	if err := msg.Msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "payload msg")
	}
	return nil
}

func (msg MsgInstantiateContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgInstantiateContract) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.WasmAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{sdk.WasmToAccAddress(senderAddr)}
}

func (msg MsgExecuteContract) Route() string {
	return RouterKey
}

func (msg MsgExecuteContract) Type() string {
	return "execute"
}

func (msg MsgExecuteContract) ValidateBasic() error {
	if _, err := sdk.WasmAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}
	if _, err := sdk.WasmAddressFromBech32(msg.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}

	if !msg.Funds.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "sentFunds")
	}
	if err := msg.Msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "payload msg")
	}
	return nil
}

func (msg MsgExecuteContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgExecuteContract) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.WasmAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{sdk.WasmToAccAddress(senderAddr)}
}

func (msg MsgExecuteContract) FnSignatureInfo() (string, int, error) {
	if err := msg.Msg.ValidateBasic(); err != nil {
		return "", 0, fmt.Errorf("failed to validate msg:%v", err)
	}

	var v interface{}
	json.Unmarshal(msg.Msg, &v)
	data := v.(map[string]interface{})
	if len(data) != 1 {
		return "", 0, fmt.Errorf("failed to check msg method:%s", string(msg.Msg.Bytes()))
	}

	method := ""
	for k, _ := range data {
		method = k
		break
	}

	if len(method) <= 0 {
		return "", 0, fmt.Errorf("msg has not method:%s", string(msg.Msg.Bytes()))
	}

	var builder strings.Builder
	builder.WriteString(msg.Contract)
	builder.WriteString(method)

	return builder.String(), 0, nil
}

func (msg MsgMigrateContract) Route() string {
	return RouterKey
}

func (msg MsgMigrateContract) Type() string {
	return "migrate"
}

func (msg MsgMigrateContract) ValidateBasic() error {
	if msg.CodeID == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "code id is required")
	}
	if _, err := sdk.WasmAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}
	if _, err := sdk.WasmAddressFromBech32(msg.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}

	if err := msg.Msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "payload msg")
	}

	return nil
}

func (msg MsgMigrateContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgMigrateContract) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.WasmAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{sdk.WasmToAccAddress(senderAddr)}
}

func (msg MsgUpdateAdmin) Route() string {
	return RouterKey
}

func (msg MsgUpdateAdmin) Type() string {
	return "update-contract-admin"
}

func (msg MsgUpdateAdmin) ValidateBasic() error {
	if _, err := sdk.WasmAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}
	if _, err := sdk.WasmAddressFromBech32(msg.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if _, err := sdk.WasmAddressFromBech32(msg.NewAdmin); err != nil {
		return sdkerrors.Wrap(err, "new admin")
	}
	if strings.EqualFold(msg.Sender, msg.NewAdmin) {
		return sdkerrors.Wrap(ErrInvalidMsg, "new admin is the same as the sender")
	}
	return nil
}

func (msg MsgUpdateAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgUpdateAdmin) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.WasmAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}

	return []sdk.AccAddress{sdk.WasmToAccAddress(senderAddr)}
}

func (msg MsgClearAdmin) Route() string {
	return RouterKey
}

func (msg MsgClearAdmin) Type() string {
	return "clear-contract-admin"
}

func (msg MsgClearAdmin) ValidateBasic() error {
	if _, err := sdk.WasmAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}
	if _, err := sdk.WasmAddressFromBech32(msg.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	return nil
}

func (msg MsgClearAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgClearAdmin) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.WasmAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{sdk.WasmToAccAddress(senderAddr)}
}

func (msg MsgIBCSend) Route() string {
	return RouterKey
}

func (msg MsgIBCSend) Type() string {
	return "wasm-ibc-send"
}

func (msg MsgIBCSend) ValidateBasic() error {
	return nil
}

func (msg MsgIBCSend) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgIBCSend) GetSigners() []sdk.AccAddress {
	return nil
}

func (msg MsgIBCCloseChannel) Route() string {
	return RouterKey
}

func (msg MsgIBCCloseChannel) Type() string {
	return "wasm-ibc-close"
}

func (msg MsgIBCCloseChannel) ValidateBasic() error {
	return nil
}

func (msg MsgIBCCloseChannel) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgIBCCloseChannel) GetSigners() []sdk.AccAddress {
	return nil
}

func (m *MsgStoreCode) CalFromAndToForPara() (string, string) {
	return strings.ToLower(ethcmm.BytesToAddress(m.GetSigners()[0]).String()[2:]), ""
}

func (m *MsgExecuteContract) CalFromAndToForPara() (string, string) {
	return strings.ToLower(ethcmm.BytesToAddress(m.GetSigners()[0]).String()[2:]), m.Contract
}
