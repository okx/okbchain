package cryptoamino

import (
	"bytes"
	"errors"
	"reflect"

	"github.com/okx/okbchain/libs/tendermint/crypto"
	"github.com/okx/okbchain/libs/tendermint/crypto/ed25519"
	"github.com/okx/okbchain/libs/tendermint/crypto/multisig"
	"github.com/okx/okbchain/libs/tendermint/crypto/secp256k1"
	"github.com/okx/okbchain/libs/tendermint/crypto/sr25519"
	"github.com/tendermint/go-amino"
)

var cdc = amino.NewCodec()

// nameTable is used to map public key concrete types back
// to their registered amino names. This should eventually be handled
// by amino. Example usage:
// nameTable[reflect.TypeOf(ed25519.PubKeyEd25519{})] = ed25519.PubKeyAminoName
var nameTable = make(map[reflect.Type]string, 3)

func init() {
	// NOTE: It's important that there be no conflicts here,
	// as that would change the canonical representations,
	// and therefore change the address.
	// TODO: Remove above note when
	// https://github.com/tendermint/go-amino/issues/9
	// is resolved
	RegisterAmino(cdc)

	// TODO: Have amino provide a way to go from concrete struct to route directly.
	// Its currently a private API
	nameTable[reflect.TypeOf(ed25519.PubKeyEd25519{})] = ed25519.PubKeyAminoName
	nameTable[reflect.TypeOf(sr25519.PubKeySr25519{})] = sr25519.PubKeyAminoName
	nameTable[reflect.TypeOf(secp256k1.PubKeySecp256k1{})] = secp256k1.PubKeyAminoName
	nameTable[reflect.TypeOf(multisig.PubKeyMultisigThreshold{})] = multisig.PubKeyMultisigThresholdAminoRoute
}

// PubkeyAminoName returns the amino route of a pubkey
// cdc is currently passed in, as eventually this will not be using
// a package level codec.
func PubkeyAminoName(cdc *amino.Codec, key crypto.PubKey) (string, bool) {
	route, found := nameTable[reflect.TypeOf(key)]
	return route, found
}

// RegisterAmino registers all crypto related types in the given (amino) codec.
func RegisterAmino(cdc *amino.Codec) {
	// These are all written here instead of
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(sr25519.PubKeySr25519{},
		sr25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)
	cdc.RegisterConcrete(multisig.PubKeyMultisigThreshold{},
		multisig.PubKeyMultisigThresholdAminoRoute, nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PrivKeyEd25519{},
		ed25519.PrivKeyAminoName, nil)
	cdc.RegisterConcrete(sr25519.PrivKeySr25519{},
		sr25519.PrivKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PrivKeySecp256k1{},
		secp256k1.PrivKeyAminoName, nil)
}

// RegisterKeyType registers an external key type to allow decoding it from bytes
func RegisterKeyType(o interface{}, name string) {
	cdc.RegisterConcrete(o, name, nil)
	nameTable[reflect.TypeOf(o)] = name
}

// PrivKeyFromBytes unmarshals private key bytes and returns a PrivKey
func PrivKeyFromBytes(privKeyBytes []byte) (privKey crypto.PrivKey, err error) {
	err = cdc.UnmarshalBinaryBare(privKeyBytes, &privKey)
	return
}

// PubKeyFromBytes unmarshals public key bytes and returns a PubKey
func PubKeyFromBytes(pubKeyBytes []byte) (pubKey crypto.PubKey, err error) {
	err = cdc.UnmarshalBinaryBare(pubKeyBytes, &pubKey)
	return
}

// hard code here for performance
var typePubKeySecp256k1Prefix = []byte{0xeb, 0x5a, 0xe9, 0x87}
var typePubKeyEd25519Prefix = []byte{0x16, 0x24, 0xde, 0x64}
var typePubKeySr25519Prefix = []byte{0x0d, 0xfb, 0x10, 0x05}

const typePrefixAndSizeLen = 4 + 1
const aminoTypePrefix = 4

// unmarshalPubKeyFromAminoFast does a fast path for amino decodes.
func unmarshalPubKeyFromAminoFast(data []byte) (crypto.PubKey, error) {
	if len(data) < aminoTypePrefix {
		return nil, errors.New("pubkey raw data size error")
	}
	if data[0] == 0x00 {
		return nil, errors.New("unmarshal pubkey with disamb do not implement")
	}

	prefix := data[0:aminoTypePrefix]
	data = data[aminoTypePrefix:]

	if bytes.Compare(typePubKeySecp256k1Prefix, prefix) == 0 {
		sub, err := amino.DecodeByteSliceWithoutCopy(&data)
		if err != nil {
			return nil, err
		}
		if len(sub) != secp256k1.PubKeySecp256k1Size && len(data) != 0 {
			return nil, errors.New("pubkey secp256k1 size error")
		}
		pubKey := secp256k1.PubKeySecp256k1{}
		copy(pubKey[:], sub)
		return pubKey, nil
	} else if bytes.Compare(typePubKeyEd25519Prefix, prefix) == 0 {
		sub, err := amino.DecodeByteSliceWithoutCopy(&data)
		if err != nil {
			return nil, err
		}
		if len(sub) != ed25519.PubKeyEd25519Size && len(data) != 0 {
			return nil, errors.New("pubkey ed25519 size error")
		}
		pubKey := ed25519.PubKeyEd25519{}
		copy(pubKey[:], sub)
		return pubKey, nil
	} else if bytes.Compare(typePubKeySr25519Prefix, prefix) == 0 {
		sub, err := amino.DecodeByteSliceWithoutCopy(&data)
		if err != nil {
			return nil, err
		}
		if len(sub) != sr25519.PubKeySr25519Size && len(data) != 0 {
			return nil, errors.New("pubkey sr25519 size error")
		}
		pubKey := sr25519.PubKeySr25519{}
		copy(pubKey[:], sub)
		return pubKey, nil
	} else {
		return nil, errors.New("unmarshal pubkey with unknown type")
	}
}

// UnmarshalPubKeyFromAmino decode pubkey from amino bytes,
// bytes should start with type prefix
func UnmarshalPubKeyFromAmino(cdc *amino.Codec, data []byte) (crypto.PubKey, error) {
	var pubkey crypto.PubKey
	var err error
	pubkey, err = unmarshalPubKeyFromAminoFast(data)
	if err != nil {
		var pubkeyTmp crypto.PubKey
		err = cdc.UnmarshalBinaryBare(data, &pubkeyTmp)
		pubkey = pubkeyTmp
	}
	return pubkey, err
}

func MarshalPubKeyToAmino(cdc *amino.Codec, key crypto.PubKey) (data []byte, err error) {
	switch key.(type) {
	case secp256k1.PubKeySecp256k1:
		data = make([]byte, 0, secp256k1.PubKeySecp256k1Size+typePrefixAndSizeLen)
		data = append(data, typePubKeySecp256k1Prefix...)
		data = append(data, byte(secp256k1.PubKeySecp256k1Size))
		keyData := key.(secp256k1.PubKeySecp256k1)
		data = append(data, keyData[:]...)
		return data, nil
	case ed25519.PubKeyEd25519:
		data = make([]byte, 0, ed25519.PubKeyEd25519Size+typePrefixAndSizeLen)
		data = append(data, typePubKeyEd25519Prefix...)
		data = append(data, byte(ed25519.PubKeyEd25519Size))
		keyData := key.(ed25519.PubKeyEd25519)
		data = append(data, keyData[:]...)
		return data, nil
	case sr25519.PubKeySr25519:
		data = make([]byte, 0, sr25519.PubKeySr25519Size+typePrefixAndSizeLen)
		data = append(data, typePubKeySr25519Prefix...)
		data = append(data, byte(sr25519.PubKeySr25519Size))
		keyData := key.(sr25519.PubKeySr25519)
		data = append(data, keyData[:]...)
		return data, nil
	}
	data, err = cdc.MarshalBinaryBare(key)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func MarshalPubKeyAminoTo(cdc *amino.Codec, key crypto.PubKey, buf *bytes.Buffer) error {
	switch keyData := key.(type) {
	case secp256k1.PubKeySecp256k1:
		buf.Write(typePubKeySecp256k1Prefix)
		buf.WriteByte(byte(secp256k1.PubKeySecp256k1Size))
		buf.Write(keyData[:])
		return nil
	case ed25519.PubKeyEd25519:
		buf.Write(typePubKeyEd25519Prefix)
		buf.WriteByte(byte(ed25519.PubKeyEd25519Size))
		buf.Write(keyData[:])
		return nil
	case sr25519.PubKeySr25519:
		buf.Write(typePubKeySr25519Prefix)
		buf.WriteByte(byte(sr25519.PubKeySr25519Size))
		buf.Write(keyData[:])
		return nil
	}
	data, err := cdc.MarshalBinaryBare(key)
	if err != nil {
		return err
	}
	buf.Write(data)
	return nil
}

func PubKeyAminoSize(pubKey crypto.PubKey, cdc *amino.Codec) int {
	switch k := pubKey.(type) {
	case secp256k1.PubKeySecp256k1:
		return amino.PrefixBytesLen + k.AminoSize(cdc)
	case ed25519.PubKeyEd25519:
		return amino.PrefixBytesLen + k.AminoSize(cdc)
	case sr25519.PubKeySr25519:
		return amino.PrefixBytesLen + k.AminoSize(cdc)
	}

	if sizer, ok := pubKey.(amino.Sizer); ok {
		var typePrefix [8]byte
		tpl, err := cdc.GetTypePrefix(pubKey, typePrefix[:])
		if err != nil {
			return 0
		}
		return tpl + sizer.AminoSize(cdc)
	} else {
		encodedPubKey, err := cdc.MarshalBinaryBare(pubKey)
		if err != nil {
			return 0
		}
		return len(encodedPubKey)
	}
}
