package common

import (
	"fmt"
	"github.com/okx/okbchain/libs/cosmos-sdk/x/auth/exported"
)

var FFF = func(count *int) func(exported.Account) bool {
	fmt.Println("\n\n fff 打印开始222")
	ret := func(account exported.Account) bool {
		*count++
		if aaaa, ok := account.(exported.ModuleAccount); ok {
			fmt.Println(aaaa.GetAddress().String(), "余额:"+aaaa.GetCoins().String(), aaaa.GetName())
		} else {
			fmt.Println(account.GetAddress().String(), "余额:"+account.GetCoins().String())
		}

		return false
	}
	return ret
}
