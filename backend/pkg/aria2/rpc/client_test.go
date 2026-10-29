package rpc

import (
	"context"
	"net"
	"testing"
	"time"
)

// aria2SkipNote explains why these integration tests skip.
const aria2SkipNote = "aria2 daemon not available on localhost:6800"

// requireAria2 skips the test when no aria2 daemon answers on localhost:6800.
//
// The client connects lazily, so New() succeeds even with nothing listening and the
// failure only surfaces on the first real call. A pre-flight probe makes the reason
// for skipping explicit instead of turning a missing dependency into a confusing
// assertion failure.
func requireAria2(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:6800", time.Second)
	if err != nil {
		t.Skipf("%s: %v", aria2SkipNote, err)
	}
	_ = conn.Close()
}

// TestHTTPAll is an integration test: it drives a real aria2 daemon. Skip when one
// is not running - see TestWebsocketCaller for the rationale.
func TestHTTPAll(t *testing.T) {
	requireAria2(t)
	const targetURL = "https://nodejs.org/dist/index.json"
	rpc, err := New(context.Background(), "http://localhost:6800/jsonrpc", "", time.Second, &DummyNotifier{})
	if err != nil {
		t.Skipf("%s: %v", aria2SkipNote, err)
	}
	defer rpc.Close()
	g, err := rpc.AddURI([]string{targetURL})
	if err != nil {
		t.Skipf("%s: %v", aria2SkipNote, err)
	}
	println(g)
	if _, err = rpc.TellActive(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.PauseAll(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellStatus(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetURIs(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetFiles(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetPeers(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellActive(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellWaiting(0, 1); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellStopped(0, 1); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetOption(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetGlobalOption(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetGlobalStat(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetSessionInfo(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.Remove(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellActive(); err != nil {
		t.Error(err)
	}
}

// TestWebsocketAll is an integration test; see TestHTTPAll.
func TestWebsocketAll(t *testing.T) {
	requireAria2(t)
	const targetURL = "https://nodejs.org/dist/index.json"
	rpc, err := New(context.Background(), "ws://localhost:6800/jsonrpc", "", time.Second, &DummyNotifier{})
	if err != nil {
		t.Skipf("%s: %v", aria2SkipNote, err)
	}
	defer rpc.Close()
	g, err := rpc.AddURI([]string{targetURL})
	if err != nil {
		t.Skipf("%s: %v", aria2SkipNote, err)
	}
	println(g)
	if _, err = rpc.TellActive(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.PauseAll(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellStatus(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetURIs(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetFiles(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetPeers(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellActive(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellWaiting(0, 1); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellStopped(0, 1); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetOption(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetGlobalOption(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetGlobalStat(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.GetSessionInfo(); err != nil {
		t.Error(err)
	}
	if _, err = rpc.Remove(g); err != nil {
		t.Error(err)
	}
	if _, err = rpc.TellActive(); err != nil {
		t.Error(err)
	}
}
