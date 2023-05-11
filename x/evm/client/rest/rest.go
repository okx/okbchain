package rest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	authrest "github.com/okx/okbchain/libs/cosmos-sdk/x/auth/client/rest"

	"github.com/okx/okbchain/x/evm/client/utils"
	"github.com/okx/okbchain/x/evm/watcher"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/rpc"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okx/okbchain/libs/cosmos-sdk/types/errors"
	"github.com/okx/okbchain/libs/cosmos-sdk/types/rest"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/types"
	tmliteProxy "github.com/okx/okbchain/libs/tendermint/lite/proxy"
	"github.com/okx/okbchain/libs/tendermint/rpc/client"
	ctypes "github.com/okx/okbchain/libs/tendermint/rpc/core/types"
	"github.com/okx/okbchain/x/common"
	evmtypes "github.com/okx/okbchain/x/evm/types"
	govRest "github.com/okx/okbchain/x/gov/client/rest"
)

// RegisterRoutes - Central function to define routes that get registered by the main application
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router) {
	r.HandleFunc("/txs/{hash}", QueryTxRequestHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/txs", authrest.QueryTxsRequestHandlerFn(cliCtx)).Methods("GET")         // default from auth
	r.HandleFunc("/txs", authrest.BroadcastTxRequest(cliCtx)).Methods("POST")              // default from auth
	r.HandleFunc("/txs/encode", authrest.EncodeTxRequestHandlerFn(cliCtx)).Methods("POST") // default from auth
	r.HandleFunc("/txs/decode", authrest.DecodeTxRequestHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc("/section", QuerySectionFn(cliCtx)).Methods("GET")
	r.HandleFunc("/contract/blocked_list", QueryContractBlockedListHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/contract/method_blocked_list", QueryContractMethodBlockedListHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/block_tx_hashes/{blockHeight}", blockTxHashesHandler(cliCtx)).Methods("GET")
	r.HandleFunc("/latestheight", latestHeightHandler(cliCtx)).Methods("GET")

	registerQueryRoutes(cliCtx, r)
}

func QueryTxRequestHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hashHexStr := vars["hash"]

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}
		var output interface{}
		output, err := QueryTx(cliCtx, hashHexStr)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return

		}

		rest.PostProcessResponseBare(w, cliCtx, output)
	}
}

// QueryTx queries for a single transaction by a hash string in hex format. An
// error is returned if the transaction does not exist or cannot be queried.
func QueryTx(cliCtx context.CLIContext, hashHexStr string) (interface{}, error) {
	// strip 0x prefix
	if strings.HasPrefix(hashHexStr, "0x") {
		hashHexStr = hashHexStr[2:]
	}

	hash, err := hex.DecodeString(hashHexStr)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	node, err := cliCtx.GetNode()
	if err != nil {
		return sdk.TxResponse{}, err
	}

	resTx, err := node.Tx(hash, !cliCtx.TrustNode)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	if !cliCtx.TrustNode {
		if err = ValidateTxResult(cliCtx, resTx); err != nil {
			return sdk.TxResponse{}, err
		}
	}

	tx, err := evmtypes.TxDecoder(cliCtx.CodecProy)(resTx.Tx, nil, resTx.Height)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	if realTx, ok := tx.(*evmtypes.MsgEthereumTx); ok {
		return getEthTxResponse(node, resTx, realTx)
	}

	// not eth Tx
	resBlocks, err := getBlocksForTxResults(cliCtx, []*ctypes.ResultTx{resTx})
	if err != nil {
		return sdk.TxResponse{}, err
	}
	var ret interface{}
	switch tx.(type) {
	case *types.IbcTx:
		jsonTx, err := types.FromProtobufTx(cliCtx.CodecProy, tx.(*types.IbcTx))
		if nil != err {
			return nil, err
		}
		return sdk.NewResponseResultTx(resTx, jsonTx, resBlocks[resTx.Height].Block.Time.Format(time.RFC3339)), nil
	default:
		ret, err = formatTxResult(cliCtx.Codec, resTx, resBlocks[resTx.Height])
	}

	return ret, err

}

func getEthTxResponse(node client.Client, resTx *ctypes.ResultTx, ethTx *evmtypes.MsgEthereumTx) (interface{}, error) {
	// Can either cache or just leave this out if not necessary
	block, err := node.Block(&resTx.Height)
	if err != nil {
		return nil, err
	}
	blockHash := ethcommon.BytesToHash(block.Block.Hash())
	height := uint64(resTx.Height)
	res, err := watcher.NewTransaction(ethTx, ethcommon.BytesToHash(resTx.Tx.Hash()), blockHash, height, uint64(resTx.Index))
	if err != nil {
		return nil, err
	}
	return json.Marshal(res)
}

// ValidateTxResult performs transaction verification.
func ValidateTxResult(cliCtx context.CLIContext, resTx *ctypes.ResultTx) error {
	if !cliCtx.TrustNode {
		check, err := cliCtx.Verify(resTx.Height)
		if err != nil {
			return err
		}
		err = resTx.Proof.Validate(check.Header.DataHash)
		if err != nil {
			return err
		}
	}
	return nil
}

func getBlocksForTxResults(cliCtx context.CLIContext, resTxs []*ctypes.ResultTx) (map[int64]*ctypes.ResultBlock, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	resBlocks := make(map[int64]*ctypes.ResultBlock)

	for _, resTx := range resTxs {
		if _, ok := resBlocks[resTx.Height]; !ok {
			resBlock, err := node.Block(&resTx.Height)
			if err != nil {
				return nil, err
			}

			resBlocks[resTx.Height] = resBlock
		}
	}

	return resBlocks, nil
}

func formatTxResult(cdc *codec.Codec, resTx *ctypes.ResultTx, resBlock *ctypes.ResultBlock) (sdk.TxResponse, error) {
	tx, err := parseTx(cdc, resTx.Tx)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	return sdk.NewResponseResultTx(resTx, tx, resBlock.Block.Time.Format(time.RFC3339)), nil
}

func parseTx(cdc *codec.Codec, txBytes []byte) (sdk.Tx, error) {
	var tx types.StdTx

	err := cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

// ManageContractDeploymentWhitelistProposalRESTHandler defines evm proposal handler
func ManageContractDeploymentWhitelistProposalRESTHandler(context.CLIContext) govRest.ProposalRESTHandler {
	return govRest.ProposalRESTHandler{}
}

// ManageContractBlockedListProposalRESTHandler defines evm proposal handler
func ManageContractBlockedListProposalRESTHandler(context.CLIContext) govRest.ProposalRESTHandler {
	return govRest.ProposalRESTHandler{}
}

// ManageContractMethodBlockedListProposalRESTHandler defines evm proposal handler
func ManageContractMethodBlockedListProposalRESTHandler(context.CLIContext) govRest.ProposalRESTHandler {
	return govRest.ProposalRESTHandler{}
}

// ManageContractBytecodeProposalRESTHandler defines evm proposal handler
func ManageContractBytecodeProposalRESTHandler(context.CLIContext) govRest.ProposalRESTHandler {
	return govRest.ProposalRESTHandler{}
}

func QuerySectionFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _, err := cliCtx.Query(fmt.Sprintf("custom/%s/%s", evmtypes.RouterKey, evmtypes.QuerySection))
		if err != nil {
			sdkErr := common.ParseSDKError(err.Error())
			common.HandleErrorMsg(w, cliCtx, sdkErr.Code, sdkErr.Message)
			return
		}

		rest.PostProcessResponseBare(w, cliCtx, res)
	}
}

// QueryContractBlockedListHandlerFn defines evm contract blocked list handler
func QueryContractBlockedListHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := fmt.Sprintf("custom/%s/%s", evmtypes.ModuleName, evmtypes.QueryContractBlockedList)

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}
		bz, _, err := cliCtx.QueryWithData(path, nil)
		if err != nil {
			common.HandleErrorResponseV2(w, http.StatusInternalServerError, common.ErrorABCIQueryFails)
			return
		}

		var blockedList evmtypes.AddressList
		cliCtx.Codec.MustUnmarshalJSON(bz, &blockedList)

		var ethAddrs []string
		for _, accAddr := range blockedList {
			ethAddrs = append(ethAddrs, ethcommon.BytesToAddress(accAddr.Bytes()).Hex())
		}

		rest.PostProcessResponseBare(w, cliCtx, ethAddrs)
	}
}

// QueryContractMethodBlockedListHandlerFn defines evm contract method blocked list handler
func QueryContractMethodBlockedListHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := fmt.Sprintf("custom/%s/%s", evmtypes.ModuleName, evmtypes.QueryContractMethodBlockedList)

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}
		bz, _, err := cliCtx.QueryWithData(path, nil)
		if err != nil {
			common.HandleErrorResponseV2(w, http.StatusInternalServerError, common.ErrorABCIQueryFails)
			return
		}

		var blockedList evmtypes.BlockedContractList
		cliCtx.Codec.MustUnmarshalJSON(bz, &blockedList)

		results := make([]utils.ResponseBlockContract, 0)
		for i, _ := range blockedList {
			ethAddr := ethcommon.BytesToAddress(blockedList[i].Address.Bytes()).Hex()
			result := utils.ResponseBlockContract{Address: ethAddr, BlockMethods: blockedList[i].BlockMethods}
			results = append(results, result)
		}

		rest.PostProcessResponseBare(w, cliCtx, results)
	}
}
func blockTxHashesHandler(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		blockHeightStr := vars["blockHeight"]
		blockHeight, err := strconv.ParseInt(blockHeightStr, 10, 64)
		if err != nil {
			common.HandleErrorMsg(w, cliCtx, common.CodeStrconvFailed, err.Error())
			return
		}
		res, err := GetBlockTxHashes(cliCtx, blockHeight)
		if err != nil {
			common.HandleErrorMsg(w, cliCtx, evmtypes.CodeGetBlockTxHashesFailed,
				fmt.Sprintf("failed to get block tx hash: %s", err.Error()))
			return
		}

		rest.PostProcessResponse(w, cliCtx, res)
	}
}
func latestHeightHandler(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h, err := rpc.GetChainHeight(cliCtx)
		if err != nil {
			common.HandleErrorMsg(w, cliCtx, evmtypes.CodeGetChainHeightFailed,
				fmt.Sprintf("failed to get chain height: %s", err.Error()))
			return
		}
		res := common.GetBaseResponse(h)
		bz, err := json.Marshal(res)
		if err != nil {
			common.HandleErrorMsg(w, cliCtx, common.CodeMarshalJSONFailed, err.Error())
		}
		rest.PostProcessResponse(w, cliCtx, bz)
	}
}

// GetBlockTxHashes return tx hashes in the block of the given height
func GetBlockTxHashes(cliCtx context.CLIContext, height int64) ([]string, error) {
	// get the node
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	// header -> BlockchainInfo
	// header, tx -> Block
	// results -> BlockResults
	res, err := node.Block(&height)
	if err != nil {
		return nil, err
	}

	if !cliCtx.TrustNode {
		check, err := cliCtx.Verify(res.Block.Height)
		if err != nil {
			return nil, err
		}

		err = tmliteProxy.ValidateBlock(res.Block, check)
		if err != nil {
			return nil, err
		}
	}

	txs := res.Block.Txs
	txLen := len(txs)
	txHashes := make([]string, txLen)
	for i, txBytes := range txs {
		txHashes[i] = fmt.Sprintf("%X", txBytes.Hash())
	}
	return txHashes, nil
}
