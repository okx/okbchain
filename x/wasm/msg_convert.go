package wasm

import (
	"encoding/json"
	"errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/baseapp"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/x/common"
	"github.com/okx/okbchain/x/wasm/types"
)

var (
	ErrCheckSignerFail = errors.New("check signer fail")
	ErrNotFindHandle   = errors.New("not find handle")
)

func init() {
	RegisterConvert()
}

func RegisterConvert() {
	baseapp.RegisterCmHandleV1("wasm/MsgStoreCode", baseapp.NewCMHandleV1(ConvertMsgStoreCode))
	baseapp.RegisterCmHandleV1("wasm/MsgInstantiateContract", baseapp.NewCMHandleV1(ConvertMsgInstantiateContract))
	baseapp.RegisterCmHandleV1("wasm/MsgExecuteContract", baseapp.NewCMHandleV1(ConvertMsgExecuteContract))
	baseapp.RegisterCmHandleV1("wasm/MsgMigrateContract", baseapp.NewCMHandleV1(ConvertMsgMigrateContract))
	baseapp.RegisterCmHandleV1("wasm/MsgUpdateAdmin", baseapp.NewCMHandleV1(ConvertMsgUpdateAdmin))
}

func ConvertMsgStoreCode(data []byte, signers []sdk.AccAddress, height int64) (sdk.Msg, error) {
	newMsg := types.MsgStoreCode{}
	err := json.Unmarshal(data, &newMsg)
	if err != nil {
		return nil, err
	}
	err = newMsg.ValidateBasic()
	if err != nil {
		return nil, err
	}
	if ok := common.CheckSignerAddress(signers, newMsg.GetSigners()); !ok {
		return nil, ErrCheckSignerFail
	}
	return &newMsg, nil
}

func ConvertMsgInstantiateContract(data []byte, signers []sdk.AccAddress, height int64) (sdk.Msg, error) {
	newMsg := types.MsgInstantiateContract{}
	err := json.Unmarshal(data, &newMsg)
	if err != nil {
		return nil, err
	}
	err = newMsg.ValidateBasic()
	if err != nil {
		return nil, err
	}
	if ok := common.CheckSignerAddress(signers, newMsg.GetSigners()); !ok {
		return nil, ErrCheckSignerFail
	}
	return &newMsg, nil
}

func ConvertMsgExecuteContract(data []byte, signers []sdk.AccAddress, height int64) (sdk.Msg, error) {
	newMsg := types.MsgExecuteContract{}
	err := json.Unmarshal(data, &newMsg)
	if err != nil {
		return nil, err
	}
	err = newMsg.ValidateBasic()
	if err != nil {
		return nil, err
	}
	if ok := common.CheckSignerAddress(signers, newMsg.GetSigners()); !ok {
		return nil, ErrCheckSignerFail
	}
	return &newMsg, nil
}

func ConvertMsgMigrateContract(data []byte, signers []sdk.AccAddress, height int64) (sdk.Msg, error) {
	newMsg := types.MsgMigrateContract{}
	err := json.Unmarshal(data, &newMsg)
	if err != nil {
		return nil, err
	}
	err = newMsg.ValidateBasic()
	if err != nil {
		return nil, err
	}
	if ok := common.CheckSignerAddress(signers, newMsg.GetSigners()); !ok {
		return nil, ErrCheckSignerFail
	}
	return &newMsg, nil
}

func ConvertMsgUpdateAdmin(data []byte, signers []sdk.AccAddress, height int64) (sdk.Msg, error) {
	newMsg := types.MsgUpdateAdmin{}
	err := json.Unmarshal(data, &newMsg)
	if err != nil {
		return nil, err
	}
	err = newMsg.ValidateBasic()
	if err != nil {
		return nil, err
	}
	if ok := common.CheckSignerAddress(signers, newMsg.GetSigners()); !ok {
		return nil, ErrCheckSignerFail
	}
	return &newMsg, nil
}
