package conf

import (
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceSnapshotIsCopied(t *testing.T) {
	values := []string{"mp4", "mkv"}
	SetSlice(VideoTypes, values)

	// Mutating the caller's slice must not affect what readers see.
	values[0] = "mutated"
	assert.Equal(t, []string{"mp4", "mkv"}, GetSlice(VideoTypes))

	SetSlice(VideoTypes, nil)
	assert.Empty(t, GetSlice(VideoTypes))
}

func TestFilenameCharMapSnapshotIsCopied(t *testing.T) {
	m := map[string]string{"/": "|"}
	SetFilenameCharMap(m)

	m["/"] = "!" // caller-side mutation must be invisible
	assert.Equal(t, "|", GetFilenameCharMap()["/"])
}

func TestPrivacyRegSnapshotIsCopied(t *testing.T) {
	reg := regexp.MustCompile("secret")
	regs := []*regexp.Regexp{reg}
	SetPrivacyReg(regs)

	regs[0] = nil
	got := GetPrivacyReg()
	assert.Len(t, got, 1)
	assert.NotNil(t, got[0])
}

// TestDerivedSettingsAreRaceFree exercises the concurrent pattern the server
// produces: handlers read while a settings save publishes. Run with -race.
func TestDerivedSettingsAreRaceFree(t *testing.T) {
	ResetDerivedSettings()
	reg := regexp.MustCompile("token")

	var wg sync.WaitGroup
	const iterations = 200

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			SetSlice(VideoTypes, []string{"mp4", "mkv"})
			SetSlice(AudioTypes, []string{"mp3"})
			SetFilenameCharMap(map[string]string{"/": "|"})
			SetPrivacyReg([]*regexp.Regexp{reg})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = GetSlice(VideoTypes)
			_ = GetSlice(AudioTypes)
			_ = GetSlice(IgnorePaths)
			_ = GetFilenameCharMap()
			_ = GetPrivacyReg()
		}
	}()
	wg.Wait()

	assert.Equal(t, []string{"mp4", "mkv"}, GetSlice(VideoTypes))
	assert.Equal(t, "|", GetFilenameCharMap()["/"])
	assert.Len(t, GetPrivacyReg(), 1)
}
