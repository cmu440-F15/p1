package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cmu440/lspnet"
	"math/rand"
	"testing"
	"time"
)

/**
 * Try to read message at server side
 *
 * @param  corrupted   true if the message will be corrupted during transmission. In this case server should not
 *                     receive any message.
 *                     false if the message will not be corrupted. In this case server is expecting expectedData
 */
func (ts *testSystem) serverTryRead(corrupted bool, expectedData []byte) {
	for {
		ts.t.Logf("server starts to read...")
		_, data, err := ts.server.Read()
		if err != nil {
			ts.t.Fatalf("Server received error during read.")
			return
		}
		if corrupted {
			ts.t.Fatalf("Server received corrupted message!")
		} else {
			close(ts.exitChan)
			if !bytes.Equal(data, expectedData) {
				ts.t.Fatalf("Expecting %s, server received message: %s", expectedData, data)
			}
		}
		return
	}
}

/* Try to read message at client side */
func (ts *testSystem) clientTryRead(corrupted bool, expectedData []byte) {
	for {
		ts.t.Logf("client starts to read...")
		data, err := ts.clients[0].Read()
		if err != nil {
			ts.t.Fatalf("Client received error during read.")
			return
		}
		if corrupted {
			ts.t.Fatalf("Client received corrupted message!")
		} else {
			close(ts.exitChan)
			if !bytes.Equal(expectedData, data) {
				ts.t.Fatalf("Expecting %s, client received message: %s", expectedData, data)
			}
		}
		return
	}
}

func randData() []byte {
	writeBytes, _ := json.Marshal(rand.Intn(100))
	return writeBytes
}

func (ts *testSystem) serverSend(data []byte) {
	err := ts.server.Write(ts.clients[0].ConnID(), data)
	if err != nil {
		ts.t.Fatalf("Error returned by server.Write(): %s", err)
	}
}

func (ts *testSystem) clientSend(data []byte) {
	err := ts.clients[0].Write(data)
	if err != nil {
		ts.t.Fatalf("Error returned by client.Write(): %s", err)
	}
}

func (ts *testSystem) testServerWithCorruptedMsg(timeout int) {
	fmt.Printf("=== %s (1 clients, 1 msgs/client, %d%% drop rate, %d window size)\n",
		ts.desc, ts.dropPercent, ts.params.WindowSize)
	data := randData()

	// First, verify that server can read uncorrupted message
	ts.t.Logf("Testing server read without data corruption")
	go ts.serverTryRead(false, data)
	go ts.clientSend(data)

	timeoutChan := time.After(time.Duration(timeout) * time.Millisecond)
	select {
	case <-timeoutChan:
		ts.t.Fatalf("Server didn't receive any message in %dms", timeout)
	case <-ts.exitChan:
	}

	// After this point, all messages will be modified during transimission.
	ts.t.Logf("Message corruption enabled, all messages will be corrupted during transimission")
	lspnet.SetMsgCorruptionPercent(100)
	defer lspnet.SetMsgCorruptionPercent(0)

	go ts.serverTryRead(true, data)
	go ts.clientSend(data)

	// If server does receive any message before timeout, your implementation is correct
	time.Sleep(time.Duration(timeout) * time.Millisecond)
}

func (ts *testSystem) testClientWithCorruptedMsg(timeout int) {
	fmt.Printf("=== %s (1 clients, 1 msgs/client, %d%% drop rate, %d window size)\n",
		ts.desc, ts.dropPercent, ts.params.WindowSize)
	data := randData()

	// First, verify that client can read uncorrupted message
	ts.t.Logf("Testing client read without data corruption")
	go ts.clientTryRead(false, data)
	go ts.serverSend(data)

	timeoutChan := time.After(time.Duration(timeout) * time.Millisecond)
	select {
	case <-timeoutChan:
		ts.t.Fatalf("Client didn't receive any message in %dms", timeout)
	case <-ts.exitChan:
	}

	// After this point, all messages will be modified during transimission.
	ts.t.Logf("Message corruption enabled, all messages will be corrupted during transimission")
	lspnet.SetMsgCorruptionPercent(100)
	defer lspnet.SetMsgCorruptionPercent(0)

	go ts.clientTryRead(true, data)
	go ts.serverSend(data)

	// If client does receive any message before timeout, your implementation is correct
	time.Sleep(time.Duration(timeout) * time.Millisecond)
}

func TestCorruptedMsgServer(t *testing.T) {
	newTestSystem(t, 1, makeParams(5, 2000, 1)).
		setDescription("TestCorruptedMsgServer: server should not receive any corrupted message").
		testServerWithCorruptedMsg(2000)
}

func TestCorruptedMsgClient(t *testing.T) {
	newTestSystem(t, 1, makeParams(5, 2000, 1)).
		setDescription("TestCorruptedMsgClient: client should not receive any corrupted message").
		testClientWithCorruptedMsg(2000)
}
