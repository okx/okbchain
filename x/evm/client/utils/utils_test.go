package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	chain "github.com/okx/okbchain/app/types"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/x/evm/types"
)

const (
	expectedManageContractDeploymentWhitelistProposalJSON = `{
  "title": "default title",
  "description": "default description",
  "distributor_addresses": [
    "ex1cftp8q8g4aa65nw9s5trwexe77d9t6cr8ndu02",
    "ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc"
  ],
  "is_added": true,
  "deposit": [
    {
      "denom": "okb",
      "amount": "100.000000000000000000"
    }
  ]
}`
	expectedManageContractBlockedListProposalJSON = `{
  "title": "default title",
  "description": "default description",
  "contract_addresses": [
    "ex1cftp8q8g4aa65nw9s5trwexe77d9t6cr8ndu02",
    "ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc"
  ],
  "is_added": true,
  "deposit": [
    {
      "denom": "okb",
      "amount": "100.000000000000000000"
    }
  ]
}`
	expectedManageContractMethodBlockedListProposalJSON = `{
  "title": "default title",
  "description": "default description",
  "contract_addresses":[
        {
            "address": "ex1cftp8q8g4aa65nw9s5trwexe77d9t6cr8ndu02",
            "block_methods": [
                {
                    "sign": "0x371303c0",
                    "extra": "inc()"
                },
                {
                    "sign": "0x579be378",
                    "extra": "onc()"
                }
            ]
        },
		{
            "address": "ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc",
            "block_methods": [
                {
                    "sign": "0x371303c0",
                    "extra": "inc()"
                },
                {
                    "sign": "0x579be378",
                    "extra": "onc()"
                }
            ]
        }
  ],
  "is_added": true,
  "deposit": [
    {
      "denom": "okb",
      "amount": "100.000000000000000000"
    }
  ]
}`
	fileName                 = "./proposal.json"
	expectedTitle            = "default title"
	expectedDescription      = "default description"
	expectedDistributorAddr1 = "ex1cftp8q8g4aa65nw9s5trwexe77d9t6cr8ndu02"
	expectedDistributorAddr2 = "ex1k0wwsg7xf9tjt3rvxdewz42e74sp286agrf9qc"
	expectedMethodSign1      = "0x371303c0"
	expectedMethodExtra1     = "inc()"
	expectedMethodSign2      = "0x579be378"
	expectedMethodExtra2     = "onc()"
)

func init() {
	config := sdk.GetConfig()
	chain.SetBech32Prefixes(config)
}

func TestParseManageContractDeploymentWhitelistProposalJSON(t *testing.T) {
	// create JSON file
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(t, err)
	_, err = f.WriteString(expectedManageContractDeploymentWhitelistProposalJSON)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// remove the temporary JSON file
	defer os.Remove(fileName)

	proposal, err := ParseManageContractDeploymentWhitelistProposalJSON(types.ModuleCdc, fileName)
	require.NoError(t, err)
	require.Equal(t, expectedTitle, proposal.Title)
	require.Equal(t, expectedDescription, proposal.Description)
	require.True(t, proposal.IsAdded)
	require.Equal(t, 1, len(proposal.Deposit))
	require.Equal(t, sdk.DefaultBondDenom, proposal.Deposit[0].Denom)
	require.True(t, sdk.NewDec(100).Equal(proposal.Deposit[0].Amount))
	require.Equal(t, 2, len(proposal.DistributorAddrs))
	require.Equal(t, expectedDistributorAddr1, proposal.DistributorAddrs[0].String())
	require.Equal(t, expectedDistributorAddr2, proposal.DistributorAddrs[1].String())
}

func TestParseManageContractBlockedListProposalJSON(t *testing.T) {
	// create JSON file
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(t, err)
	_, err = f.WriteString(expectedManageContractBlockedListProposalJSON)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// remove the temporary JSON file
	defer os.Remove(fileName)

	proposal, err := ParseManageContractBlockedListProposalJSON(types.ModuleCdc, fileName)
	require.NoError(t, err)
	require.Equal(t, expectedTitle, proposal.Title)
	require.Equal(t, expectedDescription, proposal.Description)
	require.True(t, proposal.IsAdded)
	require.Equal(t, 1, len(proposal.Deposit))
	require.Equal(t, sdk.DefaultBondDenom, proposal.Deposit[0].Denom)
	require.True(t, sdk.NewDec(100).Equal(proposal.Deposit[0].Amount))
	require.Equal(t, 2, len(proposal.ContractAddrs))
	require.Equal(t, expectedDistributorAddr1, proposal.ContractAddrs[0].String())
	require.Equal(t, expectedDistributorAddr2, proposal.ContractAddrs[1].String())
}
func TestParseManageContractMethodBlockedListProposalJSON(t *testing.T) {
	// create JSON file
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(t, err)
	_, err = f.WriteString(expectedManageContractMethodBlockedListProposalJSON)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// remove the temporary JSON file
	defer os.Remove(fileName)

	proposal, err := ParseManageContractMethodBlockedListProposalJSON(types.ModuleCdc, fileName)
	require.NoError(t, err)
	require.Equal(t, expectedTitle, proposal.Title)
	require.Equal(t, expectedDescription, proposal.Description)
	require.True(t, proposal.IsAdded)
	require.Equal(t, 1, len(proposal.Deposit))
	require.Equal(t, sdk.DefaultBondDenom, proposal.Deposit[0].Denom)
	require.True(t, sdk.NewDec(100).Equal(proposal.Deposit[0].Amount))
	require.Equal(t, 2, len(proposal.ContractList))

	methods := types.ContractMethods{
		types.ContractMethod{
			Sign:  expectedMethodSign1,
			Extra: expectedMethodExtra1,
		},
		types.ContractMethod{
			Sign:  expectedMethodSign2,
			Extra: expectedMethodExtra2,
		},
	}
	addr1, err := sdk.AccAddressFromBech32(expectedDistributorAddr1)
	require.NoError(t, err)
	addr2, err := sdk.AccAddressFromBech32(expectedDistributorAddr2)
	require.NoError(t, err)
	expectBc1 := types.NewBlockContract(addr1, methods)
	expectBc2 := types.NewBlockContract(addr2, methods)
	ok := types.BlockedContractListIsEqual(t, proposal.ContractList, types.BlockedContractList{*expectBc1, *expectBc2})
	require.True(t, ok)
}

func TestParseManageSysContractAddressProposalJSON(t *testing.T) {
	defaultSysContractAddressProposalJSON := `{
  "title":"default title",
  "description":"default description",
  "contract_address": "0xA4FFCda536CC8fF1eeFe32D32EE943b9B4e70414",
  "is_added":true,
  "deposit": [
    {
      "denom": "okb",
      "amount": "100.000000000000000000"
    }
  ]
}`
	// create JSON file
	filePathName := "./defaultSysContractAddressProposalJSON.json"
	f, err := os.OpenFile(filePathName, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(t, err)
	_, err = f.WriteString(defaultSysContractAddressProposalJSON)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// remove the temporary JSON file
	defer os.Remove(filePathName)

	proposal, err := ParseManageSysContractAddressProposalJSON(types.ModuleCdc, filePathName)
	require.NoError(t, err)
	require.Equal(t, expectedTitle, proposal.Title)
	require.Equal(t, expectedDescription, proposal.Description)
	require.True(t, proposal.IsAdded)
	require.Equal(t, 1, len(proposal.Deposit))
	require.Equal(t, sdk.DefaultBondDenom, proposal.Deposit[0].Denom)
	require.True(t, sdk.NewDec(100).Equal(proposal.Deposit[0].Amount))
}

func TestParseMangeBrczeroEVMDataProposalJSON(t *testing.T) {
	defaultBrczeroEVMDataProposalJSON := `{
  "title": "default title",
  "description": "default description",
  "tx": "f9028a018405f5e1008401c9c3808080b90237608060405234801561001057600080fd5b50610217806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c80631003e2d2146100465780632e64cec1146100625780636057361d14610080575b600080fd5b610060600480360381019061005b9190610105565b61009c565b005b61006a6100b7565b6040516100779190610141565b60405180910390f35b61009a60048036038101906100959190610105565b6100c0565b005b806000808282546100ad919061018b565b9250508190555050565b60008054905090565b8060008190555050565b600080fd5b6000819050919050565b6100e2816100cf565b81146100ed57600080fd5b50565b6000813590506100ff816100d9565b92915050565b60006020828403121561011b5761011a6100ca565b5b6000610129848285016100f0565b91505092915050565b61013b816100cf565b82525050565b60006020820190506101566000830184610132565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610196826100cf565b91506101a1836100cf565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156101d6576101d561015c565b5b82820190509291505056fea2646970667358221220318e29d6b4806f219eedd0cc861e82c13e28eb7f42161f2c780dc539b0e32b4e64736f6c634300080a00332aa01301027a081f4343c244fa06114fe3458a3fd29ff3262873bc3703efe5b885cfa0209c186a785de859af17bd7ad5d64f638ff48100f00a265a891a52c22c0ebfbe",
  "deposit": [
    {
      "denom": "okb",
      "amount": "100.000000000000000000"
    }
  ]
}`
	// create JSON file
	filePathName := "./defaultBrczeroEVMDataProposalJSON.json"
	f, err := os.OpenFile(filePathName, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(t, err)
	_, err = f.WriteString(defaultBrczeroEVMDataProposalJSON)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// remove the temporary JSON file
	defer os.Remove(filePathName)

	proposal, err := ParseManageBrczeroEVMDataProposalJSON(types.ModuleCdc, filePathName)
	require.NoError(t, err)
	require.Equal(t, expectedTitle, proposal.Title)
	require.Equal(t, expectedDescription, proposal.Description)
	//require.True(t, proposal.IsAdded)
	require.Equal(t, 1, len(proposal.Deposit))
	require.Equal(t, sdk.DefaultBondDenom, proposal.Deposit[0].Denom)
	require.True(t, sdk.NewDec(100).Equal(proposal.Deposit[0].Amount))
}
