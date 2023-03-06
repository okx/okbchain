package types

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"testing"

	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const (
	addr           = "ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc"
	addr1          = "0x83D83497431C2D3FEab296a9fba4e5FaDD2f7eD0"
	expectedOutput = `Address List:
ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc
ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc`
	expectedBlockListOutput = `BlockedContractList List:
Address: ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc
Method List:
Sign: aaaaExtra: aaaa()`
)

func TestAddressList_String(t *testing.T) {
	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)

	addrList := AddressList{accAddr, accAddr}
	require.Equal(t, expectedOutput, addrList.String())
}

func TestBlockContractList_String(t *testing.T) {
	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	bcMethod := BlockedContract{
		Address: accAddr,
		BlockMethods: ContractMethods{
			ContractMethod{
				Sign:  "aaaa",
				Extra: "aaaa()",
			},
		},
	}
	var blockContractList BlockedContractList
	blockContractList = []BlockedContract{
		bcMethod,
	}

	require.Equal(t, expectedBlockListOutput, blockContractList.String())
}

func TestBlockMethod(t *testing.T) {
	bcl := BlockedContractList{}
	accAddr1, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	accAddr2, err := sdk.AccAddressFromBech32(addr1)
	require.NoError(t, err)

	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)
	bc1 := BlockedContract{Address: accAddr1, BlockMethods: cmm}
	bc2 := BlockedContract{Address: accAddr2, BlockMethods: cmm}
	bcl = append(bcl, bc1, bc2)

	//test decode and encode
	buff := ModuleCdc.MustMarshalJSON(bcl)
	nbcl := BlockedContractList{}
	ModuleCdc.MustUnmarshalJSON(buff, &nbcl)
}

func TestContractMethodBlockedCache_SetContractMethod(t *testing.T) {
	cmbl := NewContractMethodBlockedCache()

	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)
	bc := BlockedContract{Address: accAddr, BlockMethods: cmm}

	data := ModuleCdc.MustMarshalJSON(bc.BlockMethods)
	resultBc, ok := cmbl.GetContractMethod(data)
	require.False(t, ok)
	cmbl.SetContractMethod(data, bc.BlockMethods)
	resultBc, ok = cmbl.GetContractMethod(data)
	require.True(t, ok)
	atcalData := ModuleCdc.MustMarshalJSON(resultBc)
	require.Equal(t, data, atcalData)
}

func TestContractMethodBlockedCache_GetContractMethod(t *testing.T) {
	cmbl := NewContractMethodBlockedCache()

	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)
	bc := BlockedContract{Address: accAddr, BlockMethods: cmm}

	data := ModuleCdc.MustMarshalJSON(bc.BlockMethods)
	cmbl.SetContractMethod(data, bc.BlockMethods)
	resultBc, ok := cmbl.GetContractMethod(data)
	require.True(t, ok)
	require.Equal(t, bc.BlockMethods, resultBc)
}

func TestNewBlockContract(t *testing.T) {
	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)

	//success,Address and BlockedMethods is not nil
	bc := NewBlockContract(accAddr, cmm)
	require.NotNil(t, bc)
	require.Equal(t, accAddr, bc.Address)
	require.Equal(t, cmm, bc.BlockMethods)

	//success,Address is not nil,BlockedMethods is nil
	bc = NewBlockContract(accAddr, nil)
	require.NotNil(t, bc)
	require.Equal(t, accAddr, bc.Address)
	require.Equal(t, ContractMethods{}, bc.BlockMethods)
	require.True(t, bc.IsAllMethodBlocked())

	//success,Address is not nil,BlockedMethods is nil
	bc = NewBlockContract(nil, cmm)
	require.NotNil(t, bc)
	require.Nil(t, bc.Address)
	require.Equal(t, cmm, bc.BlockMethods)

	//success,Address is  nil,BlockedMethods is nil
	bc = NewBlockContract(nil, nil)
	require.NotNil(t, bc)
	require.Nil(t, bc.Address)
	require.Equal(t, ContractMethods{}, bc.BlockMethods)
	require.True(t, bc.IsAllMethodBlocked())
}

func TestBlockedContract_IsAllMethodBlocked(t *testing.T) {
	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)

	//success,BlockedMethod is nil
	bc := NewBlockContract(accAddr, nil)
	require.True(t, bc.IsAllMethodBlocked())

	//success,BlockedMethod is make([]ContractMethod,0)
	bc = NewBlockContract(accAddr, make([]ContractMethod, 0))
	require.True(t, bc.IsAllMethodBlocked())

	//success,BlockedMethod is ContractMethods{}
	bc = NewBlockContract(accAddr, ContractMethods{})
	require.True(t, bc.IsAllMethodBlocked())

	//error,BlockedMethod is not empty
	bc = NewBlockContract(accAddr, ContractMethods{cm1})
	require.False(t, bc.IsAllMethodBlocked())
}

func TestBlockedContract_IsMethodBlocked(t *testing.T) {
	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)

	//success,bc has one method
	bc := NewBlockContract(accAddr, ContractMethods{cm1})
	require.True(t, bc.IsMethodBlocked(hexutil.Encode(method1)))

	//success,bc has multi method
	bc = NewBlockContract(accAddr, ContractMethods{cm1, cm2})
	require.True(t, bc.IsMethodBlocked(hexutil.Encode(method2)))

	//success,bc is empty
	bc = NewBlockContract(accAddr, ContractMethods{})
	require.False(t, bc.IsMethodBlocked(hexutil.Encode(method2)))

	//success,bc has not method
	bc = NewBlockContract(accAddr, ContractMethods{cm1})
	require.False(t, bc.IsMethodBlocked(hexutil.Encode(method2)))
}

func TestBlockedContract_ValidateBasic(t *testing.T) {
	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)
	bc := NewBlockContract(accAddr, cmm)

	//success
	err = bc.ValidateBasic()
	require.NoError(t, err)

	//error duplicated method
	bc = NewBlockContract(accAddr, ContractMethods{cm1, cm1})
	err = bc.ValidateBasic()
	require.Equal(t, err, ErrDuplicatedMethod)

	//error empty address
	bc = NewBlockContract(nil, ContractMethods{cm1, cm1})
	err = bc.ValidateBasic()
	require.Equal(t, err, ErrEmptyAddressBlockedContract)

	//error empty method
	emptyCM := ContractMethod{Sign: "", Extra: "test1"}
	bc = NewBlockContract(accAddr, ContractMethods{cm1, emptyCM})
	err = bc.ValidateBasic()
	require.Equal(t, err, ErrEmptyMethod)
}

func TestBlockedContractList_ValidateBasic(t *testing.T) {
	accAddr1, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm1 := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("allow")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm1 = append(cmm1, cm1, cm2)
	bc1 := NewBlockContract(accAddr1, cmm1)

	accAddr2, err := sdk.AccAddressFromBech32(addr1)
	require.NoError(t, err)
	cmm2 := ContractMethods{}
	method3 := []byte("cccc")[:4]
	method4 := []byte("dddd")[:4]
	cm3 := ContractMethod{Sign: hexutil.Encode(method3), Extra: "test3"}
	cm4 := ContractMethod{Sign: hexutil.Encode(method4), Extra: "test4"}
	cmm2 = append(cmm2, cm3, cm4)
	bc2 := NewBlockContract(accAddr2, cmm2)

	//success. blockedContractList is one item
	bcl1 := BlockedContractList{*bc1}
	require.NoError(t, bcl1.ValidateBasic())

	//success. blockedContractList is multi item
	bcl2 := BlockedContractList{*bc1, *bc2}
	require.NoError(t, bcl2.ValidateBasic())

	//error. blockedContractList is empty method
	emptyCM := ContractMethod{Sign: "", Extra: "test1"}
	bc := NewBlockContract(accAddr1, ContractMethods{cm1, emptyCM})
	bcl3 := BlockedContractList{*bc}
	require.Equal(t, ErrEmptyMethod, bcl3.ValidateBasic())

	//error. blockedContractList is empty address
	emptyCM = ContractMethod{Sign: "empty", Extra: "test1"}
	bc = NewBlockContract(nil, ContractMethods{cm1, emptyCM})
	bcl3 = BlockedContractList{*bc}
	require.Equal(t, ErrEmptyAddressBlockedContract, bcl3.ValidateBasic())

	//error. blockedContractList duplicated address
	bcl3 = BlockedContractList{*bc1, *bc1}
	require.Equal(t, ErrDuplicatedAddr, bcl3.ValidateBasic())

	//error. blockedContractList duplicated method
	bc = NewBlockContract(accAddr1, ContractMethods{cm1, cm1})
	bcl3 = BlockedContractList{*bc}
	require.Equal(t, ErrDuplicatedMethod, bcl3.ValidateBasic())
}

func TestSortContractMethods(t *testing.T) {
	cm1 := ContractMethod{Sign: "aaaa", Extra: "test1"}
	cm2 := ContractMethod{Sign: "bbbb", Extra: "test2"}
	cm21 := ContractMethod{Sign: "bbbb", Extra: "test3"}
	cm3 := ContractMethod{Sign: "cccc", Extra: "test3"}
	cm31 := ContractMethod{Sign: "cccc", Extra: "test3"}
	cm4 := ContractMethod{Sign: "dddd", Extra: "test4"}
	cm5 := ContractMethod{Sign: "eeee", Extra: "test5"}

	expected := ContractMethods{cm1, cm2, cm3, cm4, cm5}
	actual := ContractMethods{cm1, cm3, cm4, cm5, cm2}
	SortContractMethods(actual)
	require.Equal(t, expected, actual)

	expected = ContractMethods{cm1, cm2, cm21, cm3, cm4, cm5}
	actual = ContractMethods{cm1, cm3, cm4, cm5, cm2, cm21}
	SortContractMethods(actual)
	require.Equal(t, expected, actual)

	expected = ContractMethods{cm1, cm2, cm21, cm3, cm31, cm4, cm5}
	actual = ContractMethods{cm1, cm3, cm4, cm5, cm2, cm21, cm31}
	SortContractMethods(actual)
	require.Equal(t, expected, actual)

	expected = ContractMethods{cm1, cm2, cm21, cm31, cm3, cm4, cm5}
	actual = ContractMethods{cm1, cm3, cm4, cm5, cm2, cm21, cm31}
	SortContractMethods(actual)
	require.Equal(t, expected, actual)
}

func TestContractMethods_InsertContractMethods(t *testing.T) {
	method1 := hexutil.Encode([]byte("transfer")[:4])
	method2 := hexutil.Encode([]byte("allow")[:4])
	cm1 := ContractMethod{Sign: method1, Extra: "test1"}
	cm2 := ContractMethod{Sign: method2, Extra: "test2"}

	//success,insert one methods
	cm := ContractMethods{cm1, cm2}
	method3 := ContractMethod{Sign: hexutil.Encode([]byte("cccc")), Extra: "test3"}
	expected := ContractMethods{}
	expected = append(expected, cm...)
	expected = append(expected, method3)
	cm, err := cm.InsertContractMethods(ContractMethods{method3})
	require.NoError(t, err)
	SortContractMethods(expected)
	require.Equal(t, expected, cm)

	//success,insert multi methods
	cm = ContractMethods{cm1, cm2}
	method4 := ContractMethod{Sign: hexutil.Encode([]byte("dddd")), Extra: "test3"}
	method5 := ContractMethod{Sign: hexutil.Encode([]byte("eeee")), Extra: "test4"}
	expected = ContractMethods{}
	expected = append(expected, cm...)
	expected = append(expected, method4, method5)
	cm, err = cm.InsertContractMethods(ContractMethods{method4, method5})
	require.NoError(t, err)
	SortContractMethods(expected)
	require.Equal(t, expected, cm)

	//success,insert duplicated methods
	cm = ContractMethods{cm1, cm2}
	cm, err = cm.InsertContractMethods(ContractMethods{cm1})
	require.NoError(t, err)
	expected = ContractMethods{cm1, cm2}
	SortContractMethods(expected)
	require.Equal(t, expected, cm)

	//success,insert duplicated methods
	cm = ContractMethods{cm1, cm2}
	cm, err = cm.InsertContractMethods(ContractMethods{cm1})
	require.NoError(t, err)
	expected = ContractMethods{cm1, cm2}
	SortContractMethods(expected)
	require.Equal(t, expected, cm)
	//success,insert duplicated methods
	cm = ContractMethods{cm1, cm2}
	cm, err = cm.InsertContractMethods(ContractMethods{cm1, cm2})
	require.NoError(t, err)
	expected = ContractMethods{cm1, cm2}
	SortContractMethods(expected)
	require.Equal(t, expected, cm)
}
func TestContractMethods_DeleteContractMethodMap(t *testing.T) {
	method1 := hexutil.Encode([]byte("transfer")[:4])
	method2 := hexutil.Encode([]byte("allow")[:4])
	cm1 := ContractMethod{Sign: method1, Extra: "test1"}
	cm2 := ContractMethod{Sign: method2, Extra: "test2"}

	//success,delete one methods
	cm := ContractMethods{cm1, cm2}
	cm, err := cm.DeleteContractMethodMap(ContractMethods{cm2})
	require.NoError(t, err)
	expected := ContractMethods{cm1}
	SortContractMethods(expected)
	require.Equal(t, expected, cm)

	//success,delete multi methods
	cm = ContractMethods{cm1, cm2}
	cm, err = cm.DeleteContractMethodMap(ContractMethods{cm2, cm1})
	require.NoError(t, err)
	expected = ContractMethods{}
	require.Equal(t, expected, cm)

	//success,delete uncontains methods
	cm = ContractMethods{cm1, cm2}
	method3 := ContractMethod{Sign: hexutil.Encode([]byte("cccc")), Extra: "test3"}
	cm, err = cm.DeleteContractMethodMap(ContractMethods{method3})
	require.Error(t, err)
}
func TestContractMethods_GetContractMethodsMap(t *testing.T) {
	method1 := hexutil.Encode([]byte("transfer")[:4])
	method2 := hexutil.Encode([]byte("allow")[:4])
	cm1 := ContractMethod{Sign: method1, Extra: "test1"}
	cm2 := ContractMethod{Sign: method2, Extra: "test2"}
	cm := ContractMethods{cm1, cm2}
	expected := make(map[string]ContractMethod, 0)
	expected[method1] = cm1
	expected[method2] = cm2
	require.Equal(t, expected, cm.GetContractMethodsMap())
}
func TestContractMethods_IsContain(t *testing.T) {
	method1 := hexutil.Encode([]byte("transfer")[:4])
	method2 := hexutil.Encode([]byte("allow")[:4])
	cm1 := ContractMethod{Sign: method1, Extra: "test1"}
	cm2 := ContractMethod{Sign: method2, Extra: "test2"}

	//success
	cm := ContractMethods{cm1, cm2}
	require.True(t, cm.IsContain(method1))
	//error
	method3 := hexutil.Encode([]byte("cccc")[:4])
	require.False(t, cm.IsContain(method3))
}
func TestContractMethods_ValidateBasic(t *testing.T) {
	method1 := hexutil.Encode([]byte("transfer")[:4])
	method2 := hexutil.Encode([]byte("allow")[:4])
	cm1 := ContractMethod{Sign: method1, Extra: "test1"}
	cm2 := ContractMethod{Sign: method2, Extra: "test2"}

	//success
	cmm := ContractMethods{cm1, cm2}
	require.NoError(t, cmm.ValidateBasic())
	//error empty methods
	cm3 := ContractMethod{Sign: "", Extra: "test1"}
	cmm = ContractMethods{cm1, cm2, cm3}
	require.Equal(t, ErrEmptyMethod, cmm.ValidateBasic())
	//error duplicated
	cmm = ContractMethods{cm1, cm2, cm1}
	require.Equal(t, ErrDuplicatedMethod, cmm.ValidateBasic())
}

func TestBlockedContract_ValidateExtra(t *testing.T) {
	accAddr, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("approve")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm = append(cmm, cm1, cm2)
	bc := NewBlockContract(accAddr, cmm)

	//success
	err = bc.ValidateExtra()
	require.NoError(t, err)

	//error duplicated method
	bc = NewBlockContract(accAddr, ContractMethods{cm1, cm1})
	err = bc.ValidateExtra()
	require.Equal(t, err, ErrDuplicatedMethod)

	//error empty address
	bc = NewBlockContract(nil, ContractMethods{cm1, cm1})
	err = bc.ValidateExtra()
	require.Equal(t, err, ErrEmptyAddressBlockedContract)

	//error empty method
	emptyCM := ContractMethod{Sign: "", Extra: "test1"}
	bc = NewBlockContract(accAddr, ContractMethods{cm1, emptyCM})
	err = bc.ValidateExtra()
	require.Equal(t, err, ErrEmptyMethod)

	//success factor=0
	factor := GuFactor{Factor: sdk.NewDec(0)}
	bytes, err := json.Marshal(factor)
	require.NoError(t, err)
	factorCM := ContractMethod{Sign: hexutil.Encode(method2), Extra: string(bytes)}
	bc = NewBlockContract(accAddr, ContractMethods{cm1, factorCM})
	err = bc.ValidateExtra()
	require.NoError(t, err)

	//success factor>0
	factor = GuFactor{Factor: sdk.NewDec(1)}
	bytes, err = json.Marshal(factor)
	require.NoError(t, err)
	factorCM = ContractMethod{Sign: hexutil.Encode(method2), Extra: string(bytes)}
	bc = NewBlockContract(accAddr, ContractMethods{cm1, factorCM})
	err = bc.ValidateExtra()
	require.NoError(t, err)

	//err factor<0
	factor = GuFactor{Factor: sdk.NewDec(-1)}
	bytes, err = json.Marshal(factor)
	require.NoError(t, err)
	factorCM = ContractMethod{Sign: hexutil.Encode(method2), Extra: string(bytes)}
	bc = NewBlockContract(accAddr, ContractMethods{cm1, factorCM})
	err = bc.ValidateExtra()
	require.Equal(t, err, ErrGUFactor)
}

func TestBlockedContractList_ValidateExtra(t *testing.T) {
	accAddr1, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	cmm1 := ContractMethods{}
	method1 := []byte("transfer")[:4]
	method2 := []byte("allow")[:4]
	cm1 := ContractMethod{Sign: hexutil.Encode(method1), Extra: "test1"}
	cm2 := ContractMethod{Sign: hexutil.Encode(method2), Extra: "test1"}
	cmm1 = append(cmm1, cm1, cm2)
	bc1 := NewBlockContract(accAddr1, cmm1)

	accAddr2, err := sdk.AccAddressFromBech32(addr1)
	require.NoError(t, err)
	cmm2 := ContractMethods{}
	method3 := []byte("cccc")[:4]
	method4 := []byte("dddd")[:4]
	cm3 := ContractMethod{Sign: hexutil.Encode(method3), Extra: "test3"}
	cm4 := ContractMethod{Sign: hexutil.Encode(method4), Extra: "test4"}
	cmm2 = append(cmm2, cm3, cm4)
	bc2 := NewBlockContract(accAddr2, cmm2)

	//success. blockedContractList is one item
	bcl1 := BlockedContractList{*bc1}
	require.NoError(t, bcl1.ValidateExtra())

	//success. blockedContractList is multi item
	bcl2 := BlockedContractList{*bc1, *bc2}
	require.NoError(t, bcl2.ValidateExtra())

	//error. blockedContractList is empty method
	emptyCM := ContractMethod{Sign: "", Extra: "test1"}
	bc := NewBlockContract(accAddr1, ContractMethods{cm1, emptyCM})
	bcl3 := BlockedContractList{*bc}
	require.Equal(t, ErrEmptyMethod, bcl3.ValidateExtra())

	//error. blockedContractList is empty address
	emptyCM = ContractMethod{Sign: "empty", Extra: "test1"}
	bc = NewBlockContract(nil, ContractMethods{cm1, emptyCM})
	bcl3 = BlockedContractList{*bc}
	require.Equal(t, ErrEmptyAddressBlockedContract, bcl3.ValidateExtra())

	//error. blockedContractList duplicated address
	bcl3 = BlockedContractList{*bc1, *bc1}
	require.Equal(t, ErrDuplicatedAddr, bcl3.ValidateExtra())

	//error. blockedContractList duplicated method
	bc = NewBlockContract(accAddr1, ContractMethods{cm1, cm1})
	bcl3 = BlockedContractList{*bc}
	require.Equal(t, ErrDuplicatedMethod, bcl3.ValidateExtra())

	//success. factor = 0.
	factor := GuFactor{Factor: sdk.NewDec(0)}
	bytes, err := json.Marshal(factor)
	require.NoError(t, err)
	factorCM := ContractMethod{Sign: hexutil.Encode(method2), Extra: string(bytes)}
	bc = NewBlockContract(accAddr1, ContractMethods{factorCM})
	bcl2 = BlockedContractList{*bc, *bc2}
	require.NoError(t, bcl2.ValidateExtra())

	//success. factor > 0.
	factor = GuFactor{Factor: sdk.NewDec(1)}
	bytes, err = json.Marshal(factor)
	require.NoError(t, err)
	factorCM = ContractMethod{Sign: hexutil.Encode(method2), Extra: string(bytes)}
	bc = NewBlockContract(accAddr1, ContractMethods{factorCM})
	bcl2 = BlockedContractList{*bc, *bc2}
	require.NoError(t, bcl2.ValidateExtra())

	//success. factor < 0.
	factor = GuFactor{Factor: sdk.NewDec(-1)}
	bytes, err = json.Marshal(factor)
	require.NoError(t, err)
	factorCM = ContractMethod{Sign: hexutil.Encode(method2), Extra: string(bytes)}
	bc = NewBlockContract(accAddr1, ContractMethods{factorCM})
	bcl2 = BlockedContractList{*bc, *bc2}
	require.Equal(t, ErrGUFactor, bcl2.ValidateExtra())
}

func TestContractMethods_ValidateExtra(t *testing.T) {
	method1 := hexutil.Encode([]byte("transfer")[:4])
	method2 := hexutil.Encode([]byte("allow")[:4])
	cm1 := ContractMethod{Sign: method1, Extra: "test1"}
	cm2 := ContractMethod{Sign: method2, Extra: "test2"}

	//success
	cmm := ContractMethods{cm1, cm2}
	require.NoError(t, cmm.ValidateExtra())
	//error empty methods
	cm3 := ContractMethod{Sign: "", Extra: "test1"}
	cmm = ContractMethods{cm1, cm2, cm3}
	require.Equal(t, ErrEmptyMethod, cmm.ValidateExtra())
	//error duplicated
	cmm = ContractMethods{cm1, cm2, cm1}
	require.Equal(t, ErrDuplicatedMethod, cmm.ValidateExtra())

	//success. factor ==0
	factor := GuFactor{Factor: sdk.NewDec(0)}
	bytes, err := json.Marshal(factor)
	require.NoError(t, err)
	factorCM := ContractMethod{Sign: method2, Extra: string(bytes)}
	cmm = ContractMethods{cm1, factorCM}
	require.NoError(t, cmm.ValidateExtra())

	//success. factor >0
	factor = GuFactor{Factor: sdk.NewDec(1)}
	bytes, err = json.Marshal(factor)
	require.NoError(t, err)
	factorCM = ContractMethod{Sign: method2, Extra: string(bytes)}
	cmm = ContractMethods{cm1, factorCM}
	require.NoError(t, cmm.ValidateExtra())

	//error. factor <0
	factor = GuFactor{Factor: sdk.NewDec(-1)}
	bytes, err = json.Marshal(factor)
	require.NoError(t, err)
	factorCM = ContractMethod{Sign: method2, Extra: string(bytes)}
	cmm = ContractMethods{cm1, factorCM}
	require.Equal(t, ErrGUFactor, cmm.ValidateExtra())

}
