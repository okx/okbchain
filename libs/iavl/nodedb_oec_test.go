package iavl

import (
	"container/list"
	"math"
	"math/rand"
	"testing"

	db "github.com/okx/okbchain/libs/tm-db"
	"github.com/stretchr/testify/require"
)

func mockNodeDB() *nodeDB {
	memDB := db.NewMemDB()
	return newNodeDB(memDB, 10000, nil)
}

func mockNode(version int64) *Node {
	key, value := randBytes(32), randBytes(32)
	return NewNode(key, value, version)
}

func mockNodes(version int64, capacity int) (*Node, *list.List) {
	nodelist := list.New()

	root := mockNode(version)
	nodelist.PushBack(root)
	count := 1
	for count < capacity {
		lastNode := nodelist.Remove(nodelist.Front()).(*Node)
		if count < capacity {
			node := mockNode(version)
			count++
			lastNode.leftNode = node
			nodelist.PushBack(node)
		}

		if count < capacity {
			node := mockNode(version)
			count++
			lastNode.rightNode = node
			nodelist.PushBack(node)
		}
	}
	return root, nodelist
}

func Test_saveNodeToPrePersistCache(t *testing.T) {
	EnableAsyncCommit = true
	defer func() { EnableAsyncCommit = false }()

	cases := []struct {
		version  int64
		nodeNums int
	}{
		{100, 1},
		{200, 1000},
		{300, 100000},
	}

	ndb := mockNodeDB()
	for _, c := range cases {
		nodes := make([]*Node, c.nodeNums, c.nodeNums)
		for i := 0; i < c.nodeNums; i++ {
			node := mockNode(c.version)
			node.hash = node._hash()
			nodes[i] = node
			ndb.saveNodeToPrePersistCache(node)
		}

		for i := 0; i < c.nodeNums; i++ {
			require.True(t, nodes[i].prePersisted)
			require.NotNil(t, ndb.prePersistNodeCache[string(nodes[i].hash)])
		}
	}
}

func Test_batchSet(t *testing.T) {
	EnableAsyncCommit = true
	defer func() { EnableAsyncCommit = false }()

	cases := []struct {
		version  int64
		nodeNums int
	}{
		{100, 1},
		{200, 1000},
		{300, 100000},
	}
	judges := []struct {
		hashExisted  bool
		persisted    bool
		prePersisted bool
		panic        bool
	}{
		{true, true, true, false},
		{true, false, true, true},
		{false, false, true, true},
		{true, true, false, true},
		{false, true, false, true},
		{false, false, false, true},
	}

	ndb := mockNodeDB()
	for _, c := range cases {
		for _, g := range judges {

			batch := ndb.NewBatch()
			nodes := make([]*Node, c.nodeNums, c.nodeNums)
			for i := 0; i < c.nodeNums; i++ {
				nodes[i] = mockNode(c.version)
				if g.hashExisted {
					nodes[i].hash = nodes[i]._hash()
				}
				if g.persisted {
					nodes[i].persisted = true
				}
				if g.prePersisted {
					nodes[i].prePersisted = true
				}
				if g.panic {
					require.Panics(t, func() { ndb.batchSet(nodes[i], batch) })
				} else {
					require.NotPanics(t, func() { ndb.batchSet(nodes[i], batch) })
				}
			}

			if !g.panic {
				require.NoError(t, ndb.Commit(batch))

				for i := 0; i < c.nodeNums; i++ {
					require.NotNil(t, nodes[i].hash)
					get, err := ndb.dbGet(ndb.nodeKey(nodes[i].hash))
					require.NoError(t, err)
					require.NotEmpty(t, get)
				}
			}
		}
	}
}

func Test_updateBranch(t *testing.T) {
	EnableAsyncCommit = true
	defer func() { EnableAsyncCommit = false }()

	cases := []struct {
		version  int64
		nodeNums int
	}{
		{100, 1},
		{200, 10},
		{300, 100},
		{400, 1000},
		{500, 1000},
		{600, 10000},
		{700, 100000},
	}

	ndb := mockNodeDB()
	capacity := 0
	for _, c := range cases {
		capacity += c.nodeNums

		root, nodelist := mockNodes(c.version, c.nodeNums)
		ndb.updateBranch(root, map[string]*Node{})
		for elem := nodelist.Front(); elem != nil; elem = elem.Next() {
			node := elem.Value.(*Node)
			require.True(t, node.prePersisted)
			require.Nil(t, node.leftNode)
			require.Nil(t, node.rightNode)
		}
		require.Equal(t, len(ndb.prePersistNodeCache), capacity)
	}
}

func Test_updateBranchConcurrency(t *testing.T) {
	EnableAsyncCommit = true
	defer func() { EnableAsyncCommit = false }()

	cases := []struct {
		version  int64
		nodeNums int
	}{
		{100, 1},
		{200, 10},
		{300, 100},
		{400, 1000},
		{500, 1000},
		{600, 10000},
		{700, 100000},
	}

	ndb := mockNodeDB()
	capacity := 0
	for _, c := range cases {
		capacity += c.nodeNums

		root, nodelist := mockNodes(c.version, c.nodeNums)
		ndb.updateBranchConcurrency(root, map[string]*Node{})
		for elem := nodelist.Front(); elem != nil; elem = elem.Next() {
			node := elem.Value.(*Node)
			require.True(t, node.prePersisted)
			require.Nil(t, node.leftNode)
			require.Nil(t, node.rightNode)
		}
		require.Equal(t, len(ndb.prePersistNodeCache), capacity)
	}
}

func BenchmarkUpdateBranch(b *testing.B) {
	cases := []struct {
		version  int64
		nodeNums int
	}{
		{100, 1000000},
		{200, 10000000},
	}
	b.ResetTimer()
	b.Run("updateBranch", func(b *testing.B) {
		EnableAsyncCommit = true
		defer func() { EnableAsyncCommit = false }()
		b.ResetTimer()
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			ndb := mockNodeDB()
			capacity := 0
			for _, c := range cases {
				capacity += c.nodeNums
				root, _ := mockNodes(c.version, c.nodeNums)
				ndb.updateBranch(root, map[string]*Node{})
			}
		}
	})
	b.Run("updateBranchConcurrency", func(b *testing.B) {
		EnableAsyncCommit = true
		defer func() { EnableAsyncCommit = false }()
		b.ResetTimer()
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			ndb := mockNodeDB()
			capacity := 0
			for _, c := range cases {
				capacity += c.nodeNums
				root, _ := mockNodes(c.version, c.nodeNums)
				ndb.updateBranchConcurrency(root, nil)
			}
		}
	})
}

func Test_saveCommitOrphans(t *testing.T) {
	EnableAsyncCommit = true
	defer func() { EnableAsyncCommit = false }()

	cases := []struct {
		version    int64
		orphansNum int
		exist      bool
	}{
		{100, 100, true},
		{200, 1000, true},
		{300, 10000, true},
	}

	ndb := mockNodeDB()
	for n, c := range cases {
		var commitOrphans []commitOrphan
		for i := 0; i < c.orphansNum; i++ {
			node := mockNode(c.version)
			commitOrphans = append(commitOrphans, commitOrphan{Version: rand.Int63n(100) + 100*int64(n), NodeHash: node._hash()})
		}

		batch1 := ndb.NewBatch()
		batch1.Set(ndb.rootKey(c.version), []byte("root"))
		require.NoError(t, ndb.Commit(batch1))

		batch2 := ndb.NewBatch()
		ndb.saveCommitOrphans(batch2, c.version+1, commitOrphans, false)
		require.NoError(t, ndb.Commit(batch2))

		for _, orphan := range commitOrphans {
			key := ndb.orphanKey(orphan.Version, c.version, orphan.NodeHash)
			node, err := ndb.dbGet(key)
			require.NoError(t, err)
			require.Equal(t, orphan.NodeHash, node)
		}
	}
}

func Test_getNodeInTpp(t *testing.T) {
	EnableAsyncCommit = true
	defer func() { EnableAsyncCommit = false }()

	cases := []struct {
		version    int64
		orphansNum int
		exist      bool
	}{
		{100, 100, true},
		{200, 1000, true},
		{300, 10000, true},
	}

	ndb := mockNodeDB()
	for _, c := range cases {
		for i := 0; i < c.orphansNum; i++ {
			node := mockNode(c.version)
			ndb.prePersistNodeCache[string(node._hash())] = node
		}

		tpp := ndb.prePersistNodeCache
		lItem := ndb.tpp.tppVersionList.PushBack(c.version)
		ndb.tpp.tppMap[c.version] = &tppItem{
			nodeMap:  tpp,
			listItem: lItem,
		}

		for hash, node := range tpp {
			getNode, found := ndb.tpp.getNode([]byte(hash))
			require.True(t, found)
			require.EqualValues(t, node, getNode)
		}

		ndb.prePersistNodeCache = map[string]*Node{}
	}
}

func Test_getRootWithCache(t *testing.T) {
	EnableAsyncCommit = true
	defer func() { EnableAsyncCommit = false }()

	cases := []struct {
		version int64
		exist   bool
	}{
		{1, true},
		{2, true},
	}

	ndb := mockNodeDB()
	for _, c := range cases {
		rootHash := randBytes(32)
		ndb.oi.orphanItemMap[c.version] = &orphanItem{rootHash, nil}

		actualHash, ok := ndb.findRootHash(c.version)
		if c.exist {
			require.Equal(t, actualHash, rootHash)
		} else {
			require.Nil(t, actualHash)
		}
		require.Equal(t, ok, true)

		var err error
		actualHash, err = ndb.getRootWithCacheAndDB(c.version)
		if c.exist {
			require.Equal(t, actualHash, rootHash)
		} else {
			require.Nil(t, actualHash)
		}
		require.NoError(t, err)
	}
}

func Test_inVersionCacheMap(t *testing.T) {
	cases := []struct {
		version  int64
		expected bool
	}{
		{1, true},
		{2, true},
		{3, true},
		{4, true},
	}

	ndb := mockNodeDB()
	for _, c := range cases {
		rootHash := randBytes(32)
		orphanObj := &orphanItem{rootHash: rootHash}
		ndb.oi.orphanItemMap[c.version] = orphanObj
		actualHash, existed := ndb.findRootHash(c.version)
		require.Equal(t, actualHash, rootHash)
		require.Equal(t, existed, c.expected)
	}
}

func genHash(num int) []byte {
	ret := make([]byte, num)
	rand.Read(ret)
	return ret
}

func TestOrphanKeyFast(t *testing.T) {
	testCases := []struct {
		From  int64
		To    int64
		Hash  []byte
		panic bool
	}{
		{12345, 54321, genHash(32), false},
		{0, 0, genHash(32), false},
		{math.MinInt64, math.MinInt64, genHash(20), false},
		{math.MaxInt64, math.MaxInt64, genHash(10), false},
		{math.MaxInt64, math.MaxInt64, genHash(33), true},
	}

	for _, test := range testCases {
		if !test.panic {
			expect := orphanKeyFormat.Key(test.To, test.From, test.Hash)
			actual := orphanKeyFast(test.From, test.To, test.Hash)
			require.Equal(t, expect, actual)
		} else {
			require.Panics(t, func() {
				orphanKeyFormat.Key(test.To, test.From, test.Hash)
			})
			require.Panics(t, func() {
				orphanKeyFast(test.From, test.To, test.Hash)
			})
		}
	}
}

func TestFastNodeCache(t *testing.T) {
	cases := []struct {
		ndb     *nodeDB
		nodes   []*FastNode
		initFn  func(ndb *nodeDB, fast []*FastNode)
		checkFn func(ndb *nodeDB, fast []*FastNode)
	}{
		{ // getFastNodeFromCache
			ndb: mockNodeDB(),
			nodes: []*FastNode{
				{key: randBytes(32), value: randBytes(10)},
				{key: randBytes(32), value: randBytes(10)},
				{key: randBytes(2), value: randBytes(10)},
				{key: randBytes(8), value: randBytes(10)},
				{key: randBytes(64), value: randBytes(10)},
			},
			initFn: func(ndb *nodeDB, nodes []*FastNode) {
				for _, n := range nodes {
					ndb.cacheFastNode(n)
				}
			},
			checkFn: func(ndb *nodeDB, nodes []*FastNode) {
				for _, n := range nodes {
					f, ok := ndb.getFastNodeFromCache(n.key)
					require.NotNil(t, f)
					require.Equal(t, *f, *n)
					require.True(t, ok)
				}
				// add not exist check
				f, ok := ndb.getFastNodeFromCache([]byte("testkey"))
				require.Nil(t, f)
				require.False(t, ok)
			},
		},
		{ // uncacheFastNode
			ndb: mockNodeDB(),
			nodes: []*FastNode{
				{key: randBytes(32), value: randBytes(10)},
				{key: randBytes(32), value: randBytes(10)},
				{key: randBytes(2), value: randBytes(10)},
				{key: randBytes(8), value: randBytes(10)},
				{key: randBytes(64), value: randBytes(10)},
			},
			initFn: func(ndb *nodeDB, nodes []*FastNode) {
				for _, n := range nodes {
					ndb.cacheFastNode(n)
				}
				for _, n := range nodes {
					f, ok := ndb.getFastNodeFromCache(n.key)
					require.NotNil(t, f)
					require.Equal(t, *f, *n)
					require.True(t, ok)
				}
			},
			checkFn: func(ndb *nodeDB, nodes []*FastNode) {
				// handle uncache
				for _, n := range nodes[:len(nodes)/2] {
					ndb.uncacheFastNode(n.key)
					f, ok := ndb.getFastNodeFromCache(n.key)
					require.Nil(t, f)
					require.False(t, ok)
				}
				// after check
				for _, n := range nodes[len(nodes)/2:] {
					f, ok := ndb.getFastNodeFromCache(n.key)
					require.NotNil(t, f)
					require.Equal(t, *f, *n)
					require.True(t, ok)
				}
			},
		},
	}

	for _, tc := range cases {
		tc.initFn(tc.ndb, tc.nodes)
		tc.checkFn(tc.ndb, tc.nodes)
	}
}

func BenchmarkOrphanKeyFast(b *testing.B) {
	hash := genHash(32)
	var to int64 = math.MaxInt64
	var from int64 = math.MaxInt64
	b.Run("orphanKeyFormat", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			orphanKeyFormat.Key(to, from, hash)
		}
	})
	b.Run("orphanKeyFast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			orphanKeyFast(from, to, hash)
		}
	})
}
