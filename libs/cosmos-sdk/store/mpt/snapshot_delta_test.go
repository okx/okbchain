package mpt

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSnapshotDeltaMarshalUnMarshal(t *testing.T) {
	type fields struct {
		snapDestructs map[ethcmn.Hash]struct{}
		snapAccounts  map[ethcmn.Hash][]byte
		snapStorage   map[ethcmn.Hash]map[ethcmn.Hash][]byte
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"1. marshal & unmarshal", fields{snapDestructs: map[ethcmn.Hash]struct{}{ethcmn.Hash{0x01}: {}},
			snapAccounts: map[ethcmn.Hash][]byte{ethcmn.Hash{0x02}: {0x02}},
			snapStorage:  map[ethcmn.Hash]map[ethcmn.Hash][]byte{ethcmn.Hash{0x03}: {ethcmn.Hash{0x04}: {0x04}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta := &snapshotDelta{
				DeltaSnapshotDestructs:   tt.fields.snapDestructs,
				DeltaSnapshotAccounts:    tt.fields.snapAccounts,
				DeltaSnapshotSnapStorage: tt.fields.snapStorage,
			}
			data := delta.Marshal()
			assert.NotNil(t, data)
			recoverDelta := &snapshotDelta{}
			err := recoverDelta.Unmarshal(data)
			assert.NoError(t, err)
			assert.Equalf(t, recoverDelta, delta, "Marshal()")
		})
	}
}
