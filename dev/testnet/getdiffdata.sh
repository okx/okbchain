sudo rm /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node5/okbchaind/iavlaccount
sudo rm /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node1/okbchaind/iavlaccount
rm -rf ./diffdata/*

okbchaind-iavl mpt account /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node5/okbchaind/ $1 --home /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node5/okbchaind/
okbchaind mpt account /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node1/okbchaind/ $1 --home /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node1/okbchaind/

cat /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node5/okbchaind/iavlaccount | jq -r > ./diffdata/iavl-account.json
cat /Users/oker/workspace/go/src/github.com/okex/okbchain/dev/testnet/cache/node1/okbchaind/iavlaccount | jq -r > ./diffdata/mpt-account.json

