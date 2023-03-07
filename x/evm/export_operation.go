package evm

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	sdk "github.com/okx/okbchain/libs/cosmos-sdk/types"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	dbm "github.com/okx/okbchain/libs/tm-db"
	"github.com/okx/okbchain/x/evm/types"
)

const (
	codeFileSuffix    = ".code"
	storageFileSuffix = ".storage"
	codeSubPath       = "code"
	storageSubPath    = "storage"
	defaultMode       = "default"
	filesMode         = "files"
	dbMode            = "db"
)

var (
	codePath       string
	storagePath    string
	defaultPath, _ = os.Getwd()

	goroutinePool chan struct{}
	wg            sync.WaitGroup

	codeCount    uint64
	storageCount uint64

	evmByteCodeDB, evmStateDB dbm.DB
)

// initExportEnv only initializes the paths and goroutine pool
func initExportEnv(dataPath, mode string, goroutineNum uint64) {
	if dataPath == "" {
		dataPath = defaultPath
	}

	switch mode {
	case "default":
		return
	case "files":
		codePath = filepath.Join(dataPath, codeSubPath)
		storagePath = filepath.Join(dataPath, storageSubPath)

		err := os.MkdirAll(codePath, 0777)
		if err != nil {
			panic(err)
		}
		err = os.MkdirAll(storagePath, 0777)
		if err != nil {
			panic(err)
		}

		initGoroutinePool(goroutineNum)
	case "db":
		initEVMDB(dataPath)
		initGoroutinePool(goroutineNum)
	default:
		panic("unsupported export mode")
	}
}

// initImportEnv only initializes the paths and goroutine pool
func initImportEnv(dataPath, mode string, goroutineNum uint64) {
	if dataPath == "" {
		dataPath = defaultPath
	}
	switch mode {
	case "default":
		return
	case "files":
		codePath = filepath.Join(dataPath, codeSubPath)
		storagePath = filepath.Join(dataPath, storageSubPath)

		initGoroutinePool(goroutineNum)
	case "db":
		initEVMDB(dataPath)
	default:
		panic("unsupported import mode")
	}
}

// exportToFile export EVM code and storage to files
func exportToFile(ctx sdk.Context, k Keeper, address ethcmn.Address) {
	// write Code
	addGoroutine()
	go syncWriteAccountCode(ctx, k, address)
	// write Storage
	addGoroutine()
	go syncWriteAccountStorage(ctx, k, address)
}

// importFromFile import EVM code and storage from files
func importFromFile(ctx sdk.Context, logger log.Logger, k Keeper, address ethcmn.Address, codeHash []byte) {
	// read Code from file
	addGoroutine()
	go syncReadCodeFromFile(ctx, logger, k, address, codeHash)

	// read Storage From file
	addGoroutine()
	go syncReadStorageFromFile(ctx, logger, k, address)
}

// exportToDB export EVM code and storage to leveldb
func exportToDB(ctx sdk.Context, k Keeper, address ethcmn.Address, codeHash []byte) {
	if code := k.GetCode(ctx, address); len(code) > 0 {
		// TODO repeat code
		if err := evmByteCodeDB.Set(append(types.KeyPrefixCode, codeHash...), code); err != nil {
			panic(err)
		}
		codeCount++
	}

	addGoroutine()
	go exportStorage(ctx, k, address, evmStateDB)
}

// importFromDB import EVM code and storage to leveldb
func importFromDB(ctx sdk.Context, k Keeper, address ethcmn.Address, codeHash []byte) {
	if isEmptyState(evmByteCodeDB) || isEmptyState(evmStateDB) {
		panic("failed to open evm db")
	}

	code, err := evmByteCodeDB.Get(append(types.KeyPrefixCode, codeHash...))
	if err != nil {
		panic(err)
	}
	if len(code) != 0 {
		k.SetCodeDirectly(ctx, codeHash, code)
		codeCount++
	}

	prefix := types.AddressStoragePrefix(address)
	iterator, err := evmStateDB.Iterator(prefix, sdk.PrefixEndBytes(prefix))
	if err != nil {
		panic(err)
	}
	for ; iterator.Valid(); iterator.Next() {
		k.SetStateDirectly(ctx, address, ethcmn.BytesToHash(iterator.Key()[len(prefix):]), ethcmn.BytesToHash(iterator.Value()))
		storageCount++
	}
	iterator.Close()
}

func exportStorage(ctx sdk.Context, k Keeper, addr ethcmn.Address, db dbm.DB) {
	defer finishGoroutine()

	prefix := types.AddressStoragePrefix(addr)
	err := k.ForEachStorage(ctx, addr, func(key, value ethcmn.Hash) bool {
		db.Set(append(prefix, key.Bytes()...), value.Bytes())
		atomic.AddUint64(&storageCount, 1)
		return false
	})
	if err != nil {
		panic(err)
	}
}

func initEVMDB(path string) {
	var err error
	evmByteCodeDB, err = sdk.NewDB("evm_bytecode", path)
	if err != nil {
		panic(err)
	}
	evmStateDB, err = sdk.NewDB("evm_state", path)
	if err != nil {
		panic(err)
	}
}

// initGoroutinePool creates an appropriate number of maximum goroutine
func initGoroutinePool(goroutineNum uint64) {
	if goroutineNum == 0 {
		goroutineNum = uint64(runtime.NumCPU()-1) * 16
	}
	goroutinePool = make(chan struct{}, goroutineNum)
}

// addGoroutine if goroutinePool is not full, then create a goroutine
func addGoroutine() {
	goroutinePool <- struct{}{}
	wg.Add(1)
}

// finishGoroutine follows the function addGoroutine
func finishGoroutine() {
	<-goroutinePool
	wg.Done()
}

// createFile creates a file based on a absolute path
func createFile(filePath string) *os.File {
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	return file
}

// closeFile closes the current file and writer, in case of the waste of memory
func closeFile(writer *bufio.Writer, file *os.File) {
	err := writer.Flush()
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
}

// writeOneLine only writes data into one line
func writeOneLine(writer *bufio.Writer, data string) {
	_, err := writer.WriteString(data)
	if err != nil {
		panic(err)
	}
}

// ************************************************************************************************************
// the List of functions are used for writing different type of data into files
//    First, get data from cache or db
//    Second, format data, then write them into file
//    note: there is no way of adding log when ExportGenesis, because it will generate many logs in genesis.json
// ************************************************************************************************************
// syncWriteAccountCode synchronize the process of writing types.Code into individual file.
// It doesn't create file when there is no code linked to an account
func syncWriteAccountCode(ctx sdk.Context, k Keeper, address ethcmn.Address) {
	defer finishGoroutine()

	code := k.GetCode(ctx, address)
	if len(code) != 0 {
		file := createFile(filepath.Join(codePath, address.String()+codeFileSuffix))
		writer := bufio.NewWriter(file)
		defer closeFile(writer, file)
		writeOneLine(writer, hexutil.Bytes(code).String())
		atomic.AddUint64(&codeCount, 1)
	}
}

// syncWriteAccountStorage synchronize the process of writing types.Storage into individual file
// It will delete the file when there is no storage linked to a contract
func syncWriteAccountStorage(ctx sdk.Context, k Keeper, address ethcmn.Address) {
	defer finishGoroutine()

	filename := filepath.Join(storagePath, address.String()+storageFileSuffix)
	index := 0
	defer func() {
		if index == 0 { // make a judgement that there is a slice of ethtypes.State or not
			if err := os.Remove(filename); err != nil {
				panic(err)
			}
		} else {
			atomic.AddUint64(&storageCount, uint64(index))
		}
	}()

	file := createFile(filename)
	writer := bufio.NewWriter(file)
	defer closeFile(writer, file)

	// call this function, used for iterating all the key&value based on an address
	err := k.ForEachStorage(ctx, address, func(key, value ethcmn.Hash) bool {
		writeOneLine(writer, fmt.Sprintf("%s:%s\n", key.Hex(), value.Hex()))
		index++
		return false
	})
	if err != nil {
		panic(err)
	}
}

// ************************************************************************************************************
// the List of functions are used for loading different type of data, then persists data on db
//    First, get data from local file
//    Second, format data, then set them into db
// ************************************************************************************************************
// syncReadCodeFromFile synchronize the process of setting types.Code into evm db when InitGenesis
func syncReadCodeFromFile(ctx sdk.Context, logger log.Logger, k Keeper, address ethcmn.Address, codeHash []byte) {
	defer finishGoroutine()

	codeFilePath := filepath.Join(codePath, address.String()+codeFileSuffix)
	if pathExist(codeFilePath) {
		logger.Debug("start loading code", "filename", address.String()+codeFileSuffix)
		bin, err := ioutil.ReadFile(codeFilePath)
		if err != nil {
			panic(err)
		}

		// make "0x608002412.....80" string into a slice of byte
		code := hexutil.MustDecode(string(bin))

		// Set contract code into db, ignoring setting in cache
		k.SetCodeDirectly(ctx, codeHash, code)
		atomic.AddUint64(&codeCount, 1)
	}
}

// syncReadStorageFromFile synchronize the process of setting types.Storage into evm db when InitGenesis
func syncReadStorageFromFile(ctx sdk.Context, logger log.Logger, k Keeper, address ethcmn.Address) {
	defer finishGoroutine()

	storageFilePath := filepath.Join(storagePath, address.String()+storageFileSuffix)
	if pathExist(storageFilePath) {
		logger.Debug("start loading storage", "filename", address.String()+storageFileSuffix)
		f, err := os.Open(storageFilePath)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		rd := bufio.NewReader(f)
		for {
			// eg. kvStr = "0xc543bf77d2a7bddbeb14b8d8bfa3405a8410be06d8c3e68d5bd5e7b9abd43d39:0x4e584d0000000000000000000000000000000000000000000000000000000006\n"
			kvStr, err := rd.ReadString('\n')
			if err != nil || io.EOF == err {
				break
			}
			// remove '\n' in the end of string, then split kvStr based on ':'
			kvPair := strings.Split(strings.ReplaceAll(kvStr, "\n", ""), ":")
			//convert hexStr into common.Hash struct
			key, value := ethcmn.HexToHash(kvPair[0]), ethcmn.HexToHash(kvPair[1])
			// Set the state of key&value into db, ignoring setting in cache
			k.SetStateDirectly(ctx, address, key, value)
			atomic.AddUint64(&storageCount, 1)
		}
	}
}

// pathExist used for judging the file or path exist or not when InitGenesis
func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func isEmptyState(db dbm.DB) bool {
	return db.Stats()["leveldb.sstables"] == ""
}

func CloseDB() {
	evmByteCodeDB.Close()
	evmStateDB.Close()
}
