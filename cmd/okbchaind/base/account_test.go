package base

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestDecodeAccount(t *testing.T) {
	payload := ethcommon.Hex2Bytes("91ce76ed0a5b0a143e457ab645f27c54b7407e0a2d342d0d9e2f23da12120a036f6b62120b0203d05b19dcdff4bc753d1a26f3b3cd0321034bad0c5cc922d8cddbb090ee42d8f07ec6661f403cc29158f31d781e44c754c320f8b3820a28bd9d011220c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a4701a2056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	// payload := ethcommon.Hex2Bytes("0bfd7d9a0a290a14f1829676db577682e944fc3493d451b67ff3e29f120f0a036f6b62120802024ddf5bcca3c32013120d6665655f636f6c6c6563746f72")
	t.Log(string(payload))
	acc := DecodeAccount("", payload)
	t.Log(acc)
}
