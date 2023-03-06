package conn

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	ethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	amino "github.com/tendermint/go-amino"

	"github.com/okx/okbchain/libs/tendermint/libs/log"
)

const maxPingPongPacketSize = 1024 // bytes

func createTestMConnection(conn net.Conn) *MConnection {
	onReceive := func(chID byte, msgBytes []byte) {
	}
	onError := func(r interface{}) {
	}
	c := createMConnectionWithCallbacks(conn, onReceive, onError)
	c.SetLogger(log.TestingLogger())
	return c
}

func createMConnectionWithCallbacks(
	conn net.Conn,
	onReceive func(chID byte, msgBytes []byte),
	onError func(r interface{}),
) *MConnection {
	cfg := DefaultMConnConfig()
	cfg.PingInterval = 90 * time.Millisecond
	cfg.PongTimeout = 45 * time.Millisecond
	chDescs := []*ChannelDescriptor{{ID: 0x01, Priority: 1, SendQueueCapacity: 1}}
	c := NewMConnectionWithConfig(conn, chDescs, onReceive, onError, cfg)
	c.SetLogger(log.TestingLogger())
	return c
}

func TestMConnectionSendFlushStop(t *testing.T) {
	server, client := NetPipe()
	defer server.Close() // nolint: errcheck
	defer client.Close() // nolint: errcheck

	clientConn := createTestMConnection(client)
	err := clientConn.Start()
	require.Nil(t, err)
	defer clientConn.Stop()

	msg := []byte("abc")
	assert.True(t, clientConn.Send(0x01, msg))

	aminoMsgLength := 14

	// start the reader in a new routine, so we can flush
	errCh := make(chan error)
	go func() {
		msgB := make([]byte, aminoMsgLength)
		_, err := server.Read(msgB)
		if err != nil {
			t.Error(err)
			return
		}
		errCh <- err
	}()

	// stop the conn - it should flush all conns
	clientConn.FlushStop()

	timer := time.NewTimer(3 * time.Second)
	select {
	case <-errCh:
	case <-timer.C:
		t.Error("timed out waiting for msgs to be read")
	}
}

func TestMConnectionSend(t *testing.T) {
	server, client := NetPipe()
	defer server.Close() // nolint: errcheck
	defer client.Close() // nolint: errcheck

	mconn := createTestMConnection(client)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	msg := []byte("Ant-Man")
	assert.True(t, mconn.Send(0x01, msg))
	// Note: subsequent Send/TrySend calls could pass because we are reading from
	// the send queue in a separate goroutine.
	_, err = server.Read(make([]byte, len(msg)))
	if err != nil {
		t.Error(err)
	}
	assert.True(t, mconn.CanSend(0x01))

	msg = []byte("Spider-Man")
	assert.True(t, mconn.TrySend(0x01, msg))
	_, err = server.Read(make([]byte, len(msg)))
	if err != nil {
		t.Error(err)
	}

	assert.False(t, mconn.CanSend(0x05), "CanSend should return false because channel is unknown")
	assert.False(t, mconn.Send(0x05, []byte("Absorbing Man")), "Send should return false because channel is unknown")
}

func TestMConnectionReceive(t *testing.T) {
	server, client := NetPipe()
	defer server.Close() // nolint: errcheck
	defer client.Close() // nolint: errcheck

	receivedCh := make(chan []byte)
	errorsCh := make(chan interface{})
	onReceive := func(chID byte, msgBytes []byte) {
		receivedCh <- msgBytes
	}
	onError := func(r interface{}) {
		errorsCh <- r
	}
	mconn1 := createMConnectionWithCallbacks(client, onReceive, onError)
	err := mconn1.Start()
	require.Nil(t, err)
	defer mconn1.Stop()

	mconn2 := createTestMConnection(server)
	err = mconn2.Start()
	require.Nil(t, err)
	defer mconn2.Stop()

	msg := []byte("Cyclops")
	assert.True(t, mconn2.Send(0x01, msg))

	select {
	case receivedBytes := <-receivedCh:
		assert.Equal(t, msg, receivedBytes)
	case err := <-errorsCh:
		t.Fatalf("Expected %s, got %+v", msg, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Did not receive %s message in 500ms", msg)
	}
}

func TestMConnectionStatus(t *testing.T) {
	server, client := NetPipe()
	defer server.Close() // nolint: errcheck
	defer client.Close() // nolint: errcheck

	mconn := createTestMConnection(client)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	status := mconn.Status()
	assert.NotNil(t, status)
	assert.Zero(t, status.Channels[0].SendQueueSize)
}

func TestMConnectionPongTimeoutResultsInError(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	receivedCh := make(chan []byte)
	errorsCh := make(chan interface{})
	onReceive := func(chID byte, msgBytes []byte) {
		receivedCh <- msgBytes
	}
	onError := func(r interface{}) {
		errorsCh <- r
	}
	mconn := createMConnectionWithCallbacks(client, onReceive, onError)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	serverGotPing := make(chan struct{})
	go func() {
		// read ping
		var pkt PacketPing
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(server, &pkt, maxPingPongPacketSize)
		assert.Nil(t, err)
		serverGotPing <- struct{}{}
	}()
	<-serverGotPing

	pongTimerExpired := mconn.config.PongTimeout + 20*time.Millisecond
	select {
	case msgBytes := <-receivedCh:
		t.Fatalf("Expected error, but got %v", msgBytes)
	case err := <-errorsCh:
		assert.NotNil(t, err)
	case <-time.After(pongTimerExpired):
		t.Fatalf("Expected to receive error after %v", pongTimerExpired)
	}
}

func TestMConnectionMultiplePongsInTheBeginning(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	receivedCh := make(chan []byte)
	errorsCh := make(chan interface{})
	onReceive := func(chID byte, msgBytes []byte) {
		receivedCh <- msgBytes
	}
	onError := func(r interface{}) {
		errorsCh <- r
	}
	mconn := createMConnectionWithCallbacks(client, onReceive, onError)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	// sending 3 pongs in a row (abuse)
	_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPong{}))
	require.Nil(t, err)
	_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPong{}))
	require.Nil(t, err)
	_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPong{}))
	require.Nil(t, err)

	serverGotPing := make(chan struct{})
	go func() {
		// read ping (one byte)
		var (
			packet Packet
			err    error
		)
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(server, &packet, maxPingPongPacketSize)
		require.Nil(t, err)
		serverGotPing <- struct{}{}
		// respond with pong
		_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPong{}))
		require.Nil(t, err)
	}()
	<-serverGotPing

	pongTimerExpired := mconn.config.PongTimeout + 20*time.Millisecond
	select {
	case msgBytes := <-receivedCh:
		t.Fatalf("Expected no data, but got %v", msgBytes)
	case err := <-errorsCh:
		t.Fatalf("Expected no error, but got %v", err)
	case <-time.After(pongTimerExpired):
		assert.True(t, mconn.IsRunning())
	}
}

func TestMConnectionMultiplePings(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	receivedCh := make(chan []byte)
	errorsCh := make(chan interface{})
	onReceive := func(chID byte, msgBytes []byte) {
		receivedCh <- msgBytes
	}
	onError := func(r interface{}) {
		errorsCh <- r
	}
	mconn := createMConnectionWithCallbacks(client, onReceive, onError)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	// sending 3 pings in a row (abuse)
	// see https://github.com/tendermint/tendermint/issues/1190
	_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPing{}))
	require.Nil(t, err)
	var pkt PacketPong
	_, err = cdc.UnmarshalBinaryLengthPrefixedReader(server, &pkt, maxPingPongPacketSize)
	require.Nil(t, err)
	_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPing{}))
	require.Nil(t, err)
	_, err = cdc.UnmarshalBinaryLengthPrefixedReader(server, &pkt, maxPingPongPacketSize)
	require.Nil(t, err)
	_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPing{}))
	require.Nil(t, err)
	_, err = cdc.UnmarshalBinaryLengthPrefixedReader(server, &pkt, maxPingPongPacketSize)
	require.Nil(t, err)

	assert.True(t, mconn.IsRunning())
}

func TestMConnectionPingPongs(t *testing.T) {
	// check that we are not leaking any go-routines
	defer leaktest.CheckTimeout(t, 10*time.Second)()

	server, client := net.Pipe()

	defer server.Close()
	defer client.Close()

	receivedCh := make(chan []byte)
	errorsCh := make(chan interface{})
	onReceive := func(chID byte, msgBytes []byte) {
		receivedCh <- msgBytes
	}
	onError := func(r interface{}) {
		errorsCh <- r
	}
	mconn := createMConnectionWithCallbacks(client, onReceive, onError)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	serverGotPing := make(chan struct{})
	go func() {
		// read ping
		var pkt PacketPing
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(server, &pkt, maxPingPongPacketSize)
		require.Nil(t, err)
		serverGotPing <- struct{}{}
		// respond with pong
		_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPong{}))
		require.Nil(t, err)

		time.Sleep(mconn.config.PingInterval)

		// read ping
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(server, &pkt, maxPingPongPacketSize)
		require.Nil(t, err)
		// respond with pong
		_, err = server.Write(cdc.MustMarshalBinaryLengthPrefixed(PacketPong{}))
		require.Nil(t, err)
	}()
	<-serverGotPing

	pongTimerExpired := (mconn.config.PongTimeout + 20*time.Millisecond) * 2
	select {
	case msgBytes := <-receivedCh:
		t.Fatalf("Expected no data, but got %v", msgBytes)
	case err := <-errorsCh:
		t.Fatalf("Expected no error, but got %v", err)
	case <-time.After(2 * pongTimerExpired):
		assert.True(t, mconn.IsRunning())
	}
}

func TestMConnectionStopsAndReturnsError(t *testing.T) {
	server, client := NetPipe()
	defer server.Close() // nolint: errcheck
	defer client.Close() // nolint: errcheck

	receivedCh := make(chan []byte)
	errorsCh := make(chan interface{})
	onReceive := func(chID byte, msgBytes []byte) {
		receivedCh <- msgBytes
	}
	onError := func(r interface{}) {
		errorsCh <- r
	}
	mconn := createMConnectionWithCallbacks(client, onReceive, onError)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	if err := client.Close(); err != nil {
		t.Error(err)
	}

	select {
	case receivedBytes := <-receivedCh:
		t.Fatalf("Expected error, got %v", receivedBytes)
	case err := <-errorsCh:
		assert.NotNil(t, err)
		assert.False(t, mconn.IsRunning())
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Did not receive error in 500ms")
	}
}

func newClientAndServerConnsForReadErrors(t *testing.T, chOnErr chan struct{}) (*MConnection, *MConnection) {
	server, client := NetPipe()

	onReceive := func(chID byte, msgBytes []byte) {}
	onError := func(r interface{}) {}

	// create client conn with two channels
	chDescs := []*ChannelDescriptor{
		{ID: 0x01, Priority: 1, SendQueueCapacity: 1},
		{ID: 0x02, Priority: 1, SendQueueCapacity: 1},
	}
	mconnClient := NewMConnection(client, chDescs, onReceive, onError)
	mconnClient.SetLogger(log.TestingLogger().With("module", "client"))
	err := mconnClient.Start()
	require.Nil(t, err)

	// create server conn with 1 channel
	// it fires on chOnErr when there's an error
	serverLogger := log.TestingLogger().With("module", "server")
	onError = func(r interface{}) {
		chOnErr <- struct{}{}
	}
	mconnServer := createMConnectionWithCallbacks(server, onReceive, onError)
	mconnServer.SetLogger(serverLogger)
	err = mconnServer.Start()
	require.Nil(t, err)
	return mconnClient, mconnServer
}

func expectSend(ch chan struct{}) bool {
	after := time.After(time.Second * 5)
	select {
	case <-ch:
		return true
	case <-after:
		return false
	}
}

func TestMConnectionReadErrorBadEncoding(t *testing.T) {
	chOnErr := make(chan struct{})
	mconnClient, mconnServer := newClientAndServerConnsForReadErrors(t, chOnErr)
	defer mconnClient.Stop()
	defer mconnServer.Stop()

	client := mconnClient.conn

	// send badly encoded msgPacket
	bz := cdc.MustMarshalBinaryLengthPrefixed(PacketMsg{})
	bz[4] += 0x01 // Invalid prefix bytes.

	// Write it.
	_, err := client.Write(bz)
	assert.Nil(t, err)
	assert.True(t, expectSend(chOnErr), "badly encoded msgPacket")
}

func TestMConnectionReadErrorUnknownChannel(t *testing.T) {
	chOnErr := make(chan struct{})
	mconnClient, mconnServer := newClientAndServerConnsForReadErrors(t, chOnErr)
	defer mconnClient.Stop()
	defer mconnServer.Stop()

	msg := []byte("Ant-Man")

	// fail to send msg on channel unknown by client
	assert.False(t, mconnClient.Send(0x03, msg))

	// send msg on channel unknown by the server.
	// should cause an error
	assert.True(t, mconnClient.Send(0x02, msg))
	assert.True(t, expectSend(chOnErr), "unknown channel")
}

func TestMConnectionReadErrorLongMessage(t *testing.T) {
	chOnErr := make(chan struct{})
	chOnRcv := make(chan struct{})

	mconnClient, mconnServer := newClientAndServerConnsForReadErrors(t, chOnErr)
	defer mconnClient.Stop()
	defer mconnServer.Stop()

	mconnServer.onReceive = func(chID byte, msgBytes []byte) {
		chOnRcv <- struct{}{}
	}

	client := mconnClient.conn

	// send msg thats just right
	var err error
	var buf = new(bytes.Buffer)
	var packet = PacketMsg{
		ChannelID: 0x01,
		EOF:       1,
		Bytes:     make([]byte, mconnClient.config.MaxPacketMsgPayloadSize),
	}
	_, err = cdc.MarshalBinaryLengthPrefixedWriter(buf, packet)
	assert.Nil(t, err)
	_, err = client.Write(buf.Bytes())
	assert.Nil(t, err)
	assert.True(t, expectSend(chOnRcv), "msg just right")

	// send msg thats too long
	buf = new(bytes.Buffer)
	packet = PacketMsg{
		ChannelID: 0x01,
		EOF:       1,
		Bytes:     make([]byte, mconnClient.config.MaxPacketMsgPayloadSize+100),
	}
	_, err = cdc.MarshalBinaryLengthPrefixedWriter(buf, packet)
	assert.Nil(t, err)
	_, err = client.Write(buf.Bytes())
	assert.NotNil(t, err)
	assert.True(t, expectSend(chOnErr), "msg too long")
}

func TestMConnectionReadErrorUnknownMsgType(t *testing.T) {
	chOnErr := make(chan struct{})
	mconnClient, mconnServer := newClientAndServerConnsForReadErrors(t, chOnErr)
	defer mconnClient.Stop()
	defer mconnServer.Stop()

	// send msg with unknown msg type
	err := amino.EncodeUvarint(mconnClient.conn, 4)
	assert.Nil(t, err)
	_, err = mconnClient.conn.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	assert.Nil(t, err)
	assert.True(t, expectSend(chOnErr), "unknown msg type")
}

func TestMConnectionTrySend(t *testing.T) {
	server, client := NetPipe()
	defer server.Close()
	defer client.Close()

	mconn := createTestMConnection(client)
	err := mconn.Start()
	require.Nil(t, err)
	defer mconn.Stop()

	msg := []byte("Semicolon-Woman")
	resultCh := make(chan string, 2)
	assert.True(t, mconn.TrySend(0x01, msg))
	server.Read(make([]byte, len(msg)))
	assert.True(t, mconn.CanSend(0x01))
	assert.True(t, mconn.TrySend(0x01, msg))
	assert.False(t, mconn.CanSend(0x01))
	go func() {
		mconn.TrySend(0x01, msg)
		resultCh <- "TrySend"
	}()
	assert.False(t, mconn.CanSend(0x01))
	assert.False(t, mconn.TrySend(0x01, msg))
	assert.Equal(t, "TrySend", <-resultCh)
}

func TestPacketAmino(t *testing.T) {

	packets := []Packet{
		PacketPing{},
		PacketPong{},
		PacketMsg{},
		PacketMsg{0, 0, nil},
		PacketMsg{0, 0, []byte{}},
		PacketMsg{225, 225, []byte{}},
		PacketMsg{0x7f, 45, []byte{0x12, 0x34, 0x56, 0x78}},
		PacketMsg{math.MaxUint8, math.MaxUint8, []byte{0x12, 0x34, 0x56, 0x78}},
	}

	for _, packet := range packets {
		bz, err := cdc.MarshalBinaryLengthPrefixed(packet)
		require.Nil(t, err)

		nbz, err := cdc.MarshalBinaryLengthPrefixedWithRegisteredMarshaller(packet)
		require.NoError(t, err)
		require.EqualValues(t, bz, nbz)

		packet = nil
		err = cdc.UnmarshalBinaryLengthPrefixed(bz, &packet)
		require.NoError(t, err)

		v, err := cdc.UnmarshalBinaryLengthPrefixedWithRegisteredUbmarshaller(bz, &packet)
		require.NoError(t, err)
		newPacket, ok := v.(Packet)
		require.True(t, ok)

		var buf bytes.Buffer
		buf.Write(bz)
		newPacket2, n, err := unmarshalPacketFromAminoReader(&buf, int64(buf.Len()), nil)
		require.NoError(t, err)
		require.EqualValues(t, len(bz), n)

		require.EqualValues(t, packet, newPacket)
		require.EqualValues(t, packet, newPacket2)
	}
}

func BenchmarkPacketAmino(b *testing.B) {
	hash := ethcmn.BigToHash(big.NewInt(0x12345678))
	buf := bytes.Buffer{}
	for i := 0; i < 10; i++ {
		buf.Write(hash.Bytes())
	}
	msg := PacketMsg{0x7f, 45, buf.Bytes()}
	bz, err := cdc.MarshalBinaryLengthPrefixed(msg)
	require.NoError(b, err)
	b.ResetTimer()

	b.Run("ping-amino-marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var packet Packet
			packet = PacketPing{}
			_, _ = cdc.MarshalBinaryLengthPrefixed(packet)
		}
	})
	b.Run("ping-amino-marshaller", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var packet Packet
			packet = PacketPing{}
			_, _ = cdc.MarshalBinaryLengthPrefixedWithRegisteredMarshaller(packet)
		}
	})
	b.Run("msg-amino-marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var packet Packet
			packet = PacketMsg{32, 45, []byte{0x12, 0x34, 0x56, 0x78}}
			_, _ = cdc.MarshalBinaryLengthPrefixed(packet)
		}
	})
	b.Run("msg-amino-marshaller", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var packet Packet
			packet = PacketMsg{32, 45, []byte{0x12, 0x34, 0x56, 0x78}}
			_, _ = cdc.MarshalBinaryLengthPrefixedWithRegisteredMarshaller(packet)
		}
	})
	b.Run("msg-amino-unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var packet Packet
			var buf = bytes.NewBuffer(bz)
			_, err = cdc.UnmarshalBinaryLengthPrefixedReader(buf, &packet, int64(buf.Len()))
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msg-amino-unmarshaler", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var packet Packet
			var buf = bytes.NewBuffer(bz)
			packet, _, err = unmarshalPacketFromAminoReader(buf, int64(buf.Len()), nil)
			if err != nil {
				b.Fatal(err)
			}
			_ = packet
		}
	})
}

func TestBytesStringer(t *testing.T) {
	var testData = []byte("test data !!!")
	expect := fmt.Sprintf("%X", testData)
	var testStringer = bytesHexStringer(testData)
	actual := testStringer.String()
	require.EqualValues(t, expect, actual)
	actual = fmt.Sprintf("%s", testStringer)
	require.EqualValues(t, expect, actual)
}

func TestPacketMsgAmino(t *testing.T) {
	var longBytes = make([]byte, 1024)
	rand.Read(longBytes)

	testCases := []PacketMsg{
		{},
		{
			ChannelID: 12,
			EOF:       25,
		},
		{
			ChannelID: math.MaxInt8,
			EOF:       math.MaxInt8,
			Bytes:     []byte("Bytes"),
		},
		{
			Bytes: []byte{},
		},
		{
			Bytes: longBytes,
		},
	}
	for _, msg := range testCases {
		expectData, err := cdc.MarshalBinaryBare(msg)
		require.NoError(t, err)

		actualData, err := cdc.MarshalBinaryBareWithRegisteredMarshaller(msg)
		require.NoError(t, err)

		require.EqualValues(t, expectData, actualData)
		actualData, err = msg.MarshalToAmino(cdc)
		if actualData == nil {
			actualData = []byte{}
		}
		require.EqualValues(t, expectData[4:], actualData)

		require.Equal(t, len(actualData), msg.AminoSize(cdc))

		actualData, err = cdc.MarshalBinaryWithSizer(msg, false)
		require.EqualValues(t, expectData, actualData)
		require.Equal(t, getPacketMsgAminoTypePrefix(), actualData[0:4])

		expectLenPrefixData, err := cdc.MarshalBinaryLengthPrefixed(msg)
		require.NoError(t, err)
		actualLenPrefixData, err := cdc.MarshalBinaryWithSizer(msg, true)
		require.EqualValues(t, expectLenPrefixData, actualLenPrefixData)

		var expectValue PacketMsg
		err = cdc.UnmarshalBinaryBare(expectData, &expectValue)
		require.NoError(t, err)

		var actulaValue = &PacketMsg{}
		tmp, err := cdc.UnmarshalBinaryBareWithRegisteredUnmarshaller(expectData, actulaValue)
		require.NoError(t, err)
		_, ok := tmp.(*PacketMsg)
		require.True(t, ok)
		actulaValue = tmp.(*PacketMsg)

		require.EqualValues(t, expectValue, *actulaValue)
		err = actulaValue.UnmarshalFromAmino(cdc, expectData[4:])
		require.NoError(t, err)
		require.EqualValues(t, expectValue, *actulaValue)

		actulaValue = &PacketMsg{}
		err = cdc.UnmarshalBinaryLengthPrefixed(actualLenPrefixData, actulaValue)
		require.NoError(t, err)
		require.EqualValues(t, expectValue, *actulaValue)
	}
}

func Benchmark(b *testing.B) {
	var longBytes = make([]byte, 1024)
	rand.Read(longBytes)

	testCases := []PacketMsg{
		{},
		{
			ChannelID: 12,
			EOF:       25,
		},
		{
			ChannelID: math.MaxInt8,
			EOF:       math.MaxInt8,
			Bytes:     []byte("Bytes"),
		},
		{
			Bytes: []byte{},
		},
		{
			Bytes: longBytes,
		},
	}

	b.Run("amino", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, msg := range testCases {
				_, err := cdc.MarshalBinaryLengthPrefixedWithRegisteredMarshaller(&msg)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
	b.Run("sizer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, msg := range testCases {
				_, err := cdc.MarshalBinaryWithSizer(&msg, true)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

func BenchmarkMConnectionLogSendData(b *testing.B) {
	c := new(MConnection)
	chID := byte(10)
	msgBytes := []byte("Hello World!")

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "benchmark")
	var options []log.Option
	options = append(options, log.AllowInfoWith("module", "benchmark"))
	logger = log.NewFilter(logger, options...)

	c.Logger = logger
	b.ResetTimer()

	b.Run("pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.logSendData("Send", chID, msgBytes)
		}
	})

	b.Run("logger", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Debug("Send", "channel", chID, "conn", c, "msgBytes", bytesHexStringer(msgBytes))
		}
	})
}

func BenchmarkMConnectionLogReceiveMsg(b *testing.B) {
	c := new(MConnection)
	chID := byte(10)
	msgBytes := []byte("Hello World!")

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "benchmark")
	var options []log.Option
	options = append(options, log.AllowInfoWith("module", "benchmark"))
	logger = log.NewFilter(logger, options...)

	c.Logger = logger
	b.ResetTimer()

	b.Run("pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.logReceiveMsg(chID, msgBytes)
		}
	})

	b.Run("logger", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Debug("Received bytes", "chID", chID, "msgBytes", bytesHexStringer(msgBytes))
		}
	})
}

func BenchmarkChannelLogRecvPacketMsg(b *testing.B) {
	conn := new(MConnection)
	c := new(Channel)
	chID := byte(10)
	msgBytes := []byte("Hello World!")
	pk := PacketMsg{
		ChannelID: chID,
		EOF:       25,
		Bytes:     msgBytes,
	}

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "benchmark")
	var options []log.Option
	options = append(options, log.AllowInfoWith("module", "benchmark"))
	logger = log.NewFilter(logger, options...)

	c.Logger = logger
	c.conn = conn
	b.ResetTimer()

	b.Run("pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.logRecvPacketMsg(pk)
		}
	})

	b.Run("logger", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.Logger.Debug("Read PacketMsg", "conn", c.conn, "packet", pk)
		}
	})
}

func TestTimer(t *testing.T) {
	timer := time.NewTimer(1 * time.Second)
	stoped := timer.Stop()
	require.True(t, stoped)

	var timerChHashData bool
	select {
	case <-timer.C:
		timerChHashData = true
	default:
		timerChHashData = false
	}
	require.False(t, timerChHashData)

	timer.Reset(1 * time.Second)

	time.Sleep(2 * time.Second)

	stoped = timer.Stop()
	require.False(t, stoped)

	if !stoped {
		select {
		case <-timer.C:
			timerChHashData = true
		default:
			timerChHashData = false
		}
	}
	require.True(t, timerChHashData)

	timer.Reset(1 * time.Second)
	now := time.Now()
	<-timer.C
	since := time.Since(now)
	require.True(t, since > 500*time.Millisecond)

	stoped = timer.Stop()
	require.False(t, stoped)
	if !stoped {
		//_, ok := <-timer.C
		//t.Log("ok:", ok)
		select {
		case <-timer.C:
			timerChHashData = true
		default:
			timerChHashData = false
		}
	}
	require.False(t, timerChHashData)
}
