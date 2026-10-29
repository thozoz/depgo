package cmd

import (
	"testing"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHTTPServerSetsConnectionTimeouts is the regression test for a server with
// no timeouts at all: a client that dribbles header bytes forever (Slowloris) held
// a goroutine and a file descriptor per connection indefinitely.
func TestNewHTTPServerSetsConnectionTimeouts(t *testing.T) {
	oldConf := conf.Conf
	conf.Conf = conf.DefaultConfig()
	t.Cleanup(func() { conf.Conf = oldConf })

	srv := newHTTPServer("127.0.0.1:0", nil)
	require.NotNil(t, srv)

	assert.Equal(t, 20*time.Second, srv.ReadHeaderTimeout,
		"header phase must be bounded or Slowloris works")
	assert.Equal(t, 120*time.Second, srv.IdleTimeout,
		"idle keep-alive connections must not be held forever")

	// ReadTimeout and WriteTimeout must stay unset: they cover the whole body and
	// would cut long uploads and downloads mid-stream.
	assert.Zero(t, srv.ReadTimeout, "ReadTimeout would abort large uploads")
	assert.Zero(t, srv.WriteTimeout, "WriteTimeout would abort large downloads")
}

// TestNewHTTPServerFallsBackToDefaults makes sure a zeroed (or hand-edited) config
// cannot silently disable the protection.
func TestNewHTTPServerFallsBackToDefaults(t *testing.T) {
	oldConf := conf.Conf
	conf.Conf = conf.DefaultConfig()
	conf.Conf.Scheme.ReadHeaderTimeoutSeconds = 0
	conf.Conf.Scheme.IdleTimeoutSeconds = -1
	t.Cleanup(func() { conf.Conf = oldConf })

	srv := newHTTPServer("", nil)
	assert.Equal(t, 20*time.Second, srv.ReadHeaderTimeout)
	assert.Equal(t, 120*time.Second, srv.IdleTimeout)
}

// TestNewHTTPServerHonoursConfiguredValues covers operator tuning.
func TestNewHTTPServerHonoursConfiguredValues(t *testing.T) {
	oldConf := conf.Conf
	conf.Conf = conf.DefaultConfig()
	conf.Conf.Scheme.ReadHeaderTimeoutSeconds = 45
	conf.Conf.Scheme.IdleTimeoutSeconds = 300
	t.Cleanup(func() { conf.Conf = oldConf })

	srv := newHTTPServer("", nil)
	assert.Equal(t, 45*time.Second, srv.ReadHeaderTimeout)
	assert.Equal(t, 300*time.Second, srv.IdleTimeout)
}
