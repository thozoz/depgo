package handles_test

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// "Delete after reading" is an option of one link, not a property of every link.
// Reading a marked link removes the share record; the file it pointed at stays.
func TestBurnAfterReadDeletesTheShareButKeepsTheFile(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "keep-me.txt"), []byte("BURN-PAYLOAD"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "plain.txt"), []byte("PLAIN-PAYLOAD"), 0644))

	post := func(path, body string) (int, string) {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, code, msg := parseRespEnvelope(t, w)
		return code, msg
	}
	get := func(target string) (int, string) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
		return w.Code, w.Body.String()
	}
	infoCode := func(id string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/public/share/info?share_id="+id, nil))
		_, code, _ := parseRespEnvelope(t, w)
		return code
	}
	listed := func(id string) bool {
		req := httptest.NewRequest("GET", "/api/share/list?page=1&per_page=50", nil)
		req.Header.Set("Authorization", adminToken)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return strings.Contains(w.Body.String(), `"share_id":"`+id+`"`)
	}

	require.Equal(t, 200, func() int {
		c, _ := post("/api/share/create", `{"path":"/keep-me.txt","share_id":"self-destruct","burn_after_read":true}`)
		return c
	}())
	require.Equal(t, 200, func() int { c, _ := post("/api/share/create", `{"path":"/plain.txt","share_id":"ordinary"}`); return c }())

	// Opening the page and asking for metadata is not a read.
	assert.Equal(t, 200, infoCode("self-destruct"))
	assert.True(t, listed("self-destruct"))

	// A HEAD request is not a read either.
	hr := httptest.NewRecorder()
	r.ServeHTTP(hr, httptest.NewRequest("HEAD", "/sd/self-destruct", nil))
	assert.Equal(t, 200, infoCode("self-destruct"), "a HEAD request must not burn the link")

	// The first read serves the file and removes the share.
	status, body := get("/sd/self-destruct")
	require.Equal(t, 200, status)
	require.Equal(t, "BURN-PAYLOAD", body)
	assert.False(t, listed("self-destruct"), "the share must be gone from the owner's list")
	assert.Equal(t, 404, infoCode("self-destruct"))
	_, body = get("/sd/self-destruct")
	assert.NotContains(t, body, "BURN-PAYLOAD", "a second read must get nothing")

	// The file is untouched and the slug is free again.
	content, err := os.ReadFile(filepath.Join(rootDir, "keep-me.txt"))
	require.NoError(t, err)
	assert.Equal(t, "BURN-PAYLOAD", string(content))
	code, _ := post("/api/share/create", `{"path":"/keep-me.txt","share_id":"self-destruct"}`)
	assert.Equal(t, 200, code, "the slug of a burnt share can be used again")

	// A link that was not marked is never removed by reading it.
	for i := 0; i < 3; i++ {
		status, body = get("/sd/ordinary")
		require.Equal(t, 200, status)
		require.Equal(t, "PLAIN-PAYLOAD", body)
	}
	assert.True(t, listed("ordinary"))
}

// Only the flag deletes. A plain access limit of one switches the share off but
// keeps the record, as it always did.
func TestAccessLimitOfOneKeepsTheRecord(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "one.txt"), []byte("ONE"), 0644))
	req := httptest.NewRequest("POST", "/api/share/create", strings.NewReader(`{"path":"/one.txt","share_id":"limit-one","access_limit":1}`))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w))

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/sd/limit-one", nil))
	require.Equal(t, "ONE", w.Body.String())

	lr := httptest.NewRequest("GET", "/api/share/list?page=1&per_page=50", nil)
	lr.Header.Set("Authorization", adminToken)
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, lr)
	assert.Contains(t, lw.Body.String(), `"share_id":"limit-one"`, "an access limit must not delete the record")
	assert.Contains(t, lw.Body.String(), `"burn_after_read":false`)
}

// A password-protected self-destructing link burns on the read, not on the
// password check, and many simultaneous readers still get exactly one copy.
func TestBurnAfterReadWithPasswordServesOneReader(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createLimitedShare(t, r, adminToken, rootDir, "pw-burn.txt", "burnpass",
		`{"path":"/pw-burn.txt","name":"pw","password":"burnpass","burn_after_read":true}`)

	const readers = 12
	var wg sync.WaitGroup
	bodies := make([]string, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			require.NotPanics(t, func() {
				r.ServeHTTP(w, httptest.NewRequest("GET", fmt.Sprintf("/sd/%s?auth=%s", shareID, token), nil))
			})
			bodies[i] = w.Body.String()
		}(i)
	}
	wg.Wait()

	served := 0
	for _, b := range bodies {
		if strings.Contains(b, "BURN-SECRET-CONTENT") {
			served++
		} else {
			assert.NotContains(t, b, `"code":500`, "a loser of the race must get a clean answer, not a 500")
		}
	}
	assert.Equal(t, 1, served)
}

// The option can be switched on and off from the editor, and switching it off
// does not leave a one-read limit behind.
func TestBurnAfterReadCanBeToggledOnAnExistingShare(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "toggle.txt"), []byte("TOGGLE"), 0644))
	post := func(path, body string) string {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Body.String()
	}
	require.Contains(t, post("/api/share/create", `{"path":"/toggle.txt","share_id":"toggle"}`), `"code":200`)

	out := post("/api/share/update", `{"share_id":"toggle","burn_after_read":true}`)
	assert.Contains(t, out, `"burn_after_read":true`)
	assert.Contains(t, out, `"access_limit":1`)

	out = post("/api/share/update", `{"share_id":"toggle","burn_after_read":false}`)
	assert.Contains(t, out, `"burn_after_read":false`)
	assert.Contains(t, out, `"access_limit":0`, "turning it off must not leave a one-read limit")

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/sd/toggle", nil))
		require.Equal(t, "TOGGLE", w.Body.String())
	}
}
