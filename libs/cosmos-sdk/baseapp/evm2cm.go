package baseapp

import (
	"encoding/json"
	"fmt"

	sdk "github.com/okx/brczero/libs/cosmos-sdk/types"
)

var (
	cmHandles          = make(map[string]*CMHandle)
	cmHandlesV1        = make(map[string]*CMHandleV1) // used for the type of proposal milestone
	evmResultConverter func(txHash, data []byte) ([]byte, error)
	evmConvertJudge    func(msg sdk.Msg) ([]byte, bool)
	evmParamParse      func(msg sdk.Msg) ([]byte, error)
)

type MsgWrapper struct {
	Name string          `json:"type"`
	Data json.RawMessage `json:"value"`
}

type CMHandle struct {
	fn     func(data []byte, signers []sdk.AccAddress) (sdk.Msg, error)
	height int64
}

func NewCMHandle(fn func(data []byte, signers []sdk.AccAddress) (sdk.Msg, error), height int64) *CMHandle {
	return &CMHandle{
		fn:     fn,
		height: height,
	}
}

func RegisterCmHandle(msgType string, create *CMHandle) {
	if create == nil {
		panic("Register CmHandle is nil")
	}
	if _, dup := cmHandles[msgType]; dup {
		panic("Register CmHandle twice for same module and func " + msgType)
	}
	if _, dup := cmHandlesV1[msgType]; dup {
		panic("Register CmHandle have cmHandlesV1 in same module and func " + msgType)
	}
	cmHandles[msgType] = create
}

type CMHandleV1 struct {
	fn func(data []byte, signers []sdk.AccAddress, height int64) (sdk.Msg, error)
}

func NewCMHandleV1(fn func(data []byte, signers []sdk.AccAddress, height int64) (sdk.Msg, error)) *CMHandleV1 {
	return &CMHandleV1{
		fn: fn,
	}
}

func RegisterCmHandleV1(msgType string, create *CMHandleV1) {
	if create == nil {
		panic("Register CmHandleV1 is nil")
	}
	if _, dup := cmHandlesV1[msgType]; dup {
		panic("Register CmHandleV1 twice for same module and func " + msgType)
	}
	if _, dup := cmHandles[msgType]; dup {
		panic("Register CmHandleV1 have cmHandles in same module and func " + msgType)
	}
	cmHandlesV1[msgType] = create
}

func RegisterEvmResultConverter(create func(txHash, data []byte) ([]byte, error)) {
	if create == nil {
		panic("Register EvmResultConverter is nil")
	}
	evmResultConverter = create
}

func RegisterEvmParamParse(create func(msg sdk.Msg) ([]byte, error)) {
	if create == nil {
		panic("Register EvmParamParse is nil")
	}
	evmParamParse = create
}

func RegisterEvmConvertJudge(create func(msg sdk.Msg) ([]byte, bool)) {
	if create == nil {
		panic("Register EvmConvertJudge is nil")
	}
	evmConvertJudge = create
}

func ConvertMsg(msg sdk.Msg, height int64) (sdk.Msg, error) {
	v, err := evmParamParse(msg)
	if err != nil {
		return nil, err
	}
	msgWrap, err := ParseMsgWrapper(v)
	if err != nil {
		return nil, err
	}
	if cmh, ok := cmHandles[msgWrap.Name]; ok && height >= cmh.height {
		return cmh.fn(msgWrap.Data, msg.GetSigners())
	}
	if cmh, ok := cmHandlesV1[msgWrap.Name]; ok {
		return cmh.fn(msgWrap.Data, msg.GetSigners(), height)
	}
	return nil, fmt.Errorf("not find handle")
}

func ParseMsgWrapper(data []byte) (*MsgWrapper, error) {
	cmt := &MsgWrapper{}
	err := json.Unmarshal(data, cmt)
	if err != nil {
		return nil, err
	}
	if cmt.Name == "" {
		return nil, fmt.Errorf("parse msg name field is empty")
	}
	if len(cmt.Data) == 0 {
		return nil, fmt.Errorf("parse msg data field is empty")
	}
	return cmt, nil
}

func EvmResultConvert(txHash, data []byte) ([]byte, error) {
	return evmResultConverter(txHash, data)
}

func (app *BaseApp) JudgeEvmConvert(ctx sdk.Context, msg sdk.Msg) bool {
	if app.EvmSysContractAddressHandler == nil ||
		evmConvertJudge == nil ||
		evmParamParse == nil ||
		evmResultConverter == nil {
		return false
	}
	addr, ok := evmConvertJudge(msg)
	if !ok || len(addr) == 0 {
		return false
	}
	return app.EvmSysContractAddressHandler(ctx, addr)
}

func (app *BaseApp) SetEvmSysContractAddressHandler(handler sdk.EvmSysContractAddressHandler) {
	if app.sealed {
		panic("SetEvmSysContractAddressHandler() on sealed BaseApp")
	}
	app.EvmSysContractAddressHandler = handler
}
