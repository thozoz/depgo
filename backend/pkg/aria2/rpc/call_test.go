package rpc

import (
	"context"
	"testing"
	"time"
)

// TestWebsocketCaller is an integration test: it needs an aria2 daemon listening
// on localhost:6800. Skip when one is not running instead of failing, so the
// package's unit tests stay meaningful on a machine without aria2 and CI does not
// sit permanently red for a reason unrelated to the code under test.
func TestWebsocketCaller(t *testing.T) {
	time.Sleep(time.Second)
	c, err := newWebsocketCaller(context.Background(), "ws://localhost:6800/jsonrpc", time.Second, &DummyNotifier{})
	if err != nil {
		t.Skipf("aria2 daemon not available on localhost:6800: %v", err)
	}
	defer c.Close()

	var info VersionInfo
	if err := c.Call(aria2GetVersion, []interface{}{}, &info); err != nil {
		t.Error(err.Error())
	} else {
		println(info.Version)
	}
}
