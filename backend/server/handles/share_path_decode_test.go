package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createDirectoryShare shares the whole root directory and returns its id and a
// valid access token, so nested paths can be exercised.
func createDirectoryShare(t *testing.T, handler http.Handler, adminToken, password string) (shareID, token string) {
	t.Helper()
	body := fmt.Sprintf(`{"path":"/","name":"root","password":%q}`, password)
	req := httptest.NewRequest("POST", "/api/share/create", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "share create failed: %s", w.Body.String())

	var created struct {
		Data struct {
			ShareID string `json:"share_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	shareID = created.Data.ShareID
	require.NotEmpty(t, shareID)

	authBody := fmt.Sprintf(`{"share_id":%q,"password":%q}`, shareID, password)
	reqAuth := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(authBody))
	reqAuth.Header.Set("Content-Type", "application/json")
	wAuth := httptest.NewRecorder()
	handler.ServeHTTP(wAuth, reqAuth)
	require.Equal(t, 200, parseRespCode(t, wAuth), "share auth failed: %s", wAuth.Body.String())

	var authed struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wAuth.Body.Bytes(), &authed))
	token = authed.Data.Token
	require.NotEmpty(t, token)
	return shareID, token
}

// shareGet requests one file through /sd and returns the body.
func shareGet(t *testing.T, handler http.Handler, shareID, token, filename string) string {
	t.Helper()
	u := url.URL{Path: "/sd/" + shareID + "/" + filename, RawQuery: "auth=" + url.QueryEscape(token)}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", u.String(), nil))
	return w.Body.String()
}

// TestShareDownloadDoesNotDoubleDecodePath is the regression test for a
// double-decoding bug: Gin already decodes the path, so unescaping it again
// turned a literal "%25" in a filename into "%" and served a different file
// than the one requested.
func TestShareDownloadDoesNotDoubleDecodePath(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createDirectoryShare(t, r, adminToken, "rootpass")

	literalName := "report 100% done.txt"   // what the file is actually called
	encodedName := "report 100%25 done.txt" // a different, deliberately confusing file

	require.NoError(t, os.WriteFile(filepath.Join(rootDir, literalName), []byte("LITERAL-FILE"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, encodedName), []byte("PERCENT-FILE"), 0644))

	// Asking for the literal name must return the literal file.
	got := shareGet(t, r, shareID, token, literalName)
	assert.Contains(t, got, "LITERAL-FILE",
		"a filename containing %% must resolve to itself, got %q", got)
	assert.NotContains(t, got, "PERCENT-FILE", "must not resolve to the %%25 twin")

	// Asking for the "%25" name must return that file, not its decoded twin.
	gotEncoded := shareGet(t, r, shareID, token, encodedName)
	assert.Contains(t, gotEncoded, "PERCENT-FILE",
		"the %%25 name must resolve to itself, got %q", gotEncoded)
	assert.NotContains(t, gotEncoded, "LITERAL-FILE")
}

// TestShareDownloadHandlesLiteralPercentEscapeSequences makes sure a filename
// that merely looks like an escape sequence is not rejected with a decoding
// error.
func TestShareDownloadHandlesLiteralPercentEscapeSequences(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createDirectoryShare(t, r, adminToken, "rootpass")

	// A name containing %zz is not a valid escape, but it is a valid filename.
	name := "notes %zz.txt"
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, name), []byte("ODD-NAME"), 0644))

	got := shareGet(t, r, shareID, token, name)
	assert.Contains(t, got, "ODD-NAME",
		"a literal %%zz in a filename must be served, got %q", got)
}

// TestShareDownloadRejectsTraversalStill guards that removing the extra decode
// did not weaken path containment.
func TestShareDownloadRejectsTraversalStill(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createDirectoryShare(t, r, adminToken, "rootpass")

	outside := filepath.Join(filepath.Dir(rootDir), "outside-secret.txt")
	require.NoError(t, os.WriteFile(outside, []byte("OUTSIDE-SECRET"), 0644))
	t.Cleanup(func() { _ = os.Remove(outside) })

	for _, attempt := range []string{
		"../outside-secret.txt",
		"..%2Foutside-secret.txt",
		"%2e%2e%2foutside-secret.txt",
		"....//outside-secret.txt",
	} {
		t.Run(attempt, func(t *testing.T) {
			got := shareGet(t, r, shareID, token, attempt)
			assert.NotContains(t, got, "OUTSIDE-SECRET",
				"traversal attempt %q must not escape the share root, got %q", attempt, got)
		})
	}
}
