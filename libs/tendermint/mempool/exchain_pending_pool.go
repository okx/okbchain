package mempool

import (
	"strconv"
	"strings"
	"sync"

	cfg "github.com/okx/okbchain/libs/tendermint/config"
	"github.com/okx/okbchain/libs/tendermint/types"
)

const (
	FlagEnablePendingPool = "mempool.enable_pending_pool"
)

type PendingPool struct {
	maxSize         int
	addressTxsMap   map[string]map[uint64]*mempoolTx
	txsMap          map[string]*mempoolTx
	mtx             sync.RWMutex
	period          int
	reserveBlocks   int
	periodCounter   map[string]int // address with period count
	maxTxPerAddress int
}

func newPendingPool(maxSize int, period int, reserveBlocks int, maxTxPerAddress int) *PendingPool {
	return &PendingPool{
		maxSize:         maxSize,
		addressTxsMap:   make(map[string]map[uint64]*mempoolTx),
		txsMap:          make(map[string]*mempoolTx),
		period:          period,
		reserveBlocks:   reserveBlocks,
		periodCounter:   make(map[string]int),
		maxTxPerAddress: maxTxPerAddress,
	}
}

func (p *PendingPool) Size() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return len(p.txsMap)
}

func (p *PendingPool) GetWrappedAddressTxsMap() map[string]map[string]types.WrappedMempoolTx {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	wrappedAddressTxsMap := make(map[string]map[string]types.WrappedMempoolTx)
	for address, subMap := range p.addressTxsMap {
		nonceTxsMap := make(map[string]types.WrappedMempoolTx)
		for nonce, memTxPtr := range subMap {
			nonceStr := strconv.Itoa(int(nonce))
			nonceTxsMap[nonceStr] = memTxPtr.ToWrappedMempoolTx()
		}
		wrappedAddressTxsMap[address] = nonceTxsMap
	}
	return wrappedAddressTxsMap
}

func (p *PendingPool) txCount(address string) int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	if _, ok := p.addressTxsMap[address]; !ok {
		return 0
	}
	return len(p.addressTxsMap[address])
}

func (p *PendingPool) getTx(address string, nonce uint64) *mempoolTx {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	if _, ok := p.addressTxsMap[address]; ok {
		return p.addressTxsMap[address][nonce]
	}
	return nil
}

func (p *PendingPool) hasTx(tx types.Tx) bool {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	_, exist := p.txsMap[txID(tx)]
	return exist
}

func (p *PendingPool) addTx(pendingTx *mempoolTx) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	blacklist := strings.Split(cfg.DynamicConfig.GetPendingPoolBlacklist(), ",")
	// When cfg.DynamicConfig.GetPendingPoolBlacklist() == "", blacklist == []string{""} and len(blacklist) == 1.
	// Above case should be avoided.
	for _, address := range blacklist {
		if address != "" && pendingTx.from == address {
			return
		}
	}
	if _, ok := p.addressTxsMap[pendingTx.from]; !ok {
		p.addressTxsMap[pendingTx.from] = make(map[uint64]*mempoolTx)
	}
	p.addressTxsMap[pendingTx.from][pendingTx.realTx.GetNonce()] = pendingTx
	p.txsMap[txID(pendingTx.tx)] = pendingTx
}

func (p *PendingPool) removeTx(address string, nonce uint64) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if _, ok := p.addressTxsMap[address]; ok {
		if pendingTx, ok := p.addressTxsMap[address][nonce]; ok {
			delete(p.addressTxsMap[address], nonce)
			delete(p.txsMap, txID(pendingTx.tx))
		}
		if len(p.addressTxsMap[address]) == 0 {
			delete(p.addressTxsMap, address)
			delete(p.periodCounter, address)
		}
		// update period counter
		if count, ok := p.periodCounter[address]; ok && count > 0 {
			p.periodCounter[address] = count - 1
		}

	}

}

func (p *PendingPool) removeTxByHash(txHash string) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if pendingTx, ok := p.txsMap[txHash]; ok {
		delete(p.txsMap, txHash)
		if _, ok := p.addressTxsMap[pendingTx.from]; ok {
			delete(p.addressTxsMap[pendingTx.from], pendingTx.realTx.GetNonce())
			if len(p.addressTxsMap[pendingTx.from]) == 0 {
				delete(p.addressTxsMap, pendingTx.from)
				delete(p.periodCounter, pendingTx.from)
			}
			// update period counter
			if count, ok := p.periodCounter[pendingTx.from]; ok && count > 0 {
				p.periodCounter[pendingTx.from] = count - 1
			}
		}
	}
}

func (p *PendingPool) handlePendingTx(addressNonce map[string]uint64) map[string]uint64 {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	addrMap := make(map[string]uint64)
	for addr, accountNonce := range addressNonce {
		if txsMap, ok := p.addressTxsMap[addr]; ok {
			for nonce, pendingTx := range txsMap {
				// remove invalid pending tx
				if nonce <= accountNonce {
					delete(p.addressTxsMap[addr], nonce)
					delete(p.txsMap, txID(pendingTx.tx))
				} else if nonce == accountNonce+1 {
					addrMap[addr] = nonce
				}
			}
			if len(p.addressTxsMap[addr]) == 0 {
				delete(p.addressTxsMap, addr)
			}
		}
	}
	return addrMap
}

func (p *PendingPool) handlePeriodCounter() {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	for addr, txMap := range p.addressTxsMap {
		count := p.periodCounter[addr]
		if count >= p.reserveBlocks {
			delete(p.addressTxsMap, addr)
			for _, pendingTx := range txMap {
				delete(p.txsMap, txID(pendingTx.tx))
			}
			delete(p.periodCounter, addr)
		} else {
			p.periodCounter[addr] = count + 1
		}
	}
}

func (p *PendingPool) validate(address string, tx types.Tx) error {
	// tx already in pending pool
	if p.hasTx(tx) {
		return ErrTxAlreadyInPendingPool{
			txHash: txID(tx),
		}
	}

	poolSize := p.Size()
	if poolSize >= p.maxSize {
		return ErrPendingPoolIsFull{
			size:    poolSize,
			maxSize: p.maxSize,
		}
	}
	txCount := p.txCount(address)
	if txCount >= p.maxTxPerAddress {
		return ErrPendingPoolAddressLimit{
			address: address,
			size:    txCount,
			maxSize: p.maxTxPerAddress,
		}
	}
	return nil
}

type AccountRetriever interface {
	GetAccountNonce(address string) uint64
}
