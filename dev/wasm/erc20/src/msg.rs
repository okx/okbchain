use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::Uint128;
use cosmwasm_std::{CosmosMsg,CustomMsg};
use schemars::gen::SchemaGenerator;
use schemars::schema::Schema;

#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema)]
pub struct InitialBalance {
    pub address: String,
    pub amount: Uint128,
}

#[derive(Serialize, Deserialize, JsonSchema)]
pub struct InstantiateMsg {
    pub name: String,
    pub symbol: String,
    pub decimals: u8,
    pub initial_balances: Vec<InitialBalance>,
}

#[derive(Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    Approve {
        spender: String,
        amount: Uint128,
    },
    Transfer {
        recipient: String,
        amount: Uint128,
    },
    TransferFrom {
        owner: String,
        recipient: String,
        amount: Uint128,
    },
    Burn {
        amount: Uint128,
    },
    MintCW20 {
        recipient: String,
        amount: Uint128,
    },
    SendToEvm {
        evmContract: String,
        recipient: String,
        amount: Uint128,
    },
    CallToEvm {
        evmContract: String,
        calldata: String,
        value: Uint128,
    }
}

#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema, Debug)]
#[serde(rename_all = "snake_case")]
pub struct SendToEvmMsg {
    pub sender: String,
    pub contract: String,
    pub recipient: String,
    pub amount: Uint128,

}
impl Into<CosmosMsg<SendToEvmMsg>> for SendToEvmMsg {
    fn into(self) -> CosmosMsg<SendToEvmMsg> {
        CosmosMsg::Custom(self)
    }
}
impl CustomMsg for SendToEvmMsg {}

#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema, Debug)]
#[serde(rename_all = "snake_case")]
pub struct CallToEvmMsg {
    pub sender: String,
    pub evmaddr: String,
    pub calldata: String,
    pub value: Uint128,

}
impl Into<CosmosMsg<CallToEvmMsg>> for CallToEvmMsg {
    fn into(self) -> CosmosMsg<CallToEvmMsg> {
        CosmosMsg::Custom(self)
    }
}
impl CustomMsg for CallToEvmMsg {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    Balance { address: String },
    Allowance { owner: String, spender: String },
}

#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema)]
pub struct BalanceResponse {
    pub balance: Uint128,
}

#[derive(Serialize, Deserialize, Clone, PartialEq, JsonSchema)]
pub struct AllowanceResponse {
    pub allowance: Uint128,
}
