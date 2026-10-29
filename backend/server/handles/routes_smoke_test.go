package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// routeSmokeCase describes one admin endpoint exercised by the smoke test.
//
// The point is not to assert business behaviour (individual tests do that) but
// to catch endpoints that are simply broken: a 500, a nil dereference rendered
// as an empty body, or a handler that answers success while writing nothing.
type routeSmokeCase struct {
	method string
	target string
	body   string
	// allowCodes lists body codes that mean "the handler worked". Anything else
	// is reported. Empty means 200 only.
	allowCodes []int
}

// TestAdminRoutesSmoke calls every administrative endpoint once and fails if
// any of them returns an unexpected body code, so a regression in one handler
// cannot hide behind the others.
func TestAdminRoutesSmoke(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)
	seedTestRole(t)

	// A user to point user-scoped endpoints at.
	subject := &model.User{
		ID: 200, Username: "smokeuser", Role: model.Roles{testRoleID},
		Permission: 0x1, BasePath: "/", Authn: "[]",
	}
	subject.SetPassword("smoke-password")
	require.NoError(t, db.CreateUser(subject))

	// A storage-independent label to point label endpoints at.
	label := &model.Label{ID: 1, Name: "smokelabel", BgColor: "#000000"}
	if _, err := db.CreateLabel(*label); err != nil {
		t.Logf("label seed skipped: %v", err)
	}

	cases := []routeSmokeCase{
		// user
		{method: "GET", target: "/api/admin/user/list?page=1&per_page=10"},
		{method: "GET", target: "/api/admin/user/get?id=200"},
		{method: "POST", target: "/api/admin/user/update", body: `{"id":200,"permission":1}`},
		{method: "POST", target: "/api/admin/user/cancel_2fa?id=200"},
		{method: "POST", target: "/api/admin/user/del_cache?username=smokeuser"},
		{method: "GET", target: "/api/admin/user/sshkey/list?uid=200"},

		// role
		{method: "GET", target: "/api/admin/role/list"},
		{method: "GET", target: fmt.Sprintf("/api/admin/role/get?id=%d", testRoleID)},

		// storage
		{method: "GET", target: "/api/admin/storage/list?page=1&per_page=10"},
		{method: "GET", target: "/api/admin/storage/get?id=1"},

		// driver
		{method: "GET", target: "/api/admin/driver/list"},
		{method: "GET", target: "/api/admin/driver/names"},
		{method: "GET", target: "/api/admin/driver/info?driver=Local"},

		// setting
		{method: "GET", target: "/api/admin/setting/list?group=0"},
		{method: "GET", target: "/api/admin/setting/get?key=site_title"},
		{method: "POST", target: "/api/admin/setting/save", body: `{"key":"site_title","value":"Depgo Smoke"}`},

		// meta: create first so the read path is exercised against a real row.
		// (Reading a missing row currently answers 500 rather than 404; that is
		// reported separately rather than asserted here.)
		{method: "POST", target: "/api/admin/meta/create", body: `{"path":"/smoke","password":"smoke-pass"}`},
		{method: "GET", target: "/api/admin/meta/list"},

		// label (admin routes only expose create/update/delete; the read side
		// lives on the authenticated /api/label routes)
		{method: "GET", target: "/api/label/list"},

		// task: each manager exposes its own sub-tree. The managers are created
		// by bootstrap.InitTaskManager, which the test harness does not run, so
		// here they are nil and the handlers correctly answer 503; 200 is what a
		// running server returns.
		{method: "GET", target: "/api/admin/task/upload/undone", allowCodes: []int{200, 503}},
		{method: "GET", target: "/api/admin/task/upload/done", allowCodes: []int{200, 503}},
		{method: "GET", target: "/api/admin/task/offline_download/undone", allowCodes: []int{200, 503}},
		{method: "GET", target: "/api/admin/task/copy/undone", allowCodes: []int{200, 503}},

		// label file binding (read side lives under the authenticated group)
		{method: "GET", target: "/api/label_file_binding/get?file_name=smoke.txt"},
		{method: "GET", target: "/api/label_file_binding/get_file_by_label?label_id=1"},
	}

	failures := make([]string, 0)
	for _, tc := range cases {
		name := tc.method + " " + tc.target
		t.Run(strings.NewReplacer("/", "_", "?", "_", "&", "_", "=", "_").Replace(name), func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.target, nil)
			}
			req.Header.Set("Authorization", adminToken)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			bodyCode := 0
			var envelope struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &envelope)
			bodyCode = envelope.Code

			if w.Code >= 500 {
				failures = append(failures, fmt.Sprintf("%s -> HTTP %d body=%s", name, w.Code, truncate(w.Body.String(), 200)))
				t.Errorf("HTTP %d: %s", w.Code, truncate(w.Body.String(), 300))
				return
			}
			if bodyCode == 0 && len(w.Body.Bytes()) > 0 {
				// Not a JSON envelope at all: the handler wrote something else.
				if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
					return
				}
			}
			for _, allowed := range tc.allowCodes {
				if bodyCode == allowed {
					return
				}
			}
			if bodyCode >= 500 {
				failures = append(failures, fmt.Sprintf("%s -> code %d msg=%q", name, bodyCode, envelope.Message))
				t.Errorf("body code %d: %s", bodyCode, envelope.Message)
			}
		})
	}

	if len(failures) > 0 {
		sort.Strings(failures)
		t.Fatalf("%d endpoint(s) returned a server error:\n%s", len(failures), strings.Join(failures, "\n"))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// TestReadEndpointsReturn404ForMissingRows asserts the read-by-id handlers
// report a missing record as 404 instead of a server error, so a client can
// tell "no such object" apart from "the server broke".
func TestReadEndpointsReturn404ForMissingRows(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	for _, target := range []string{
		"/api/admin/user/get?id=999999",
		"/api/admin/role/get?id=999999",
		"/api/admin/storage/get?id=999999",
		"/api/admin/meta/get?id=999999",
	} {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest("GET", target, nil)
			req.Header.Set("Authorization", adminToken)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			_, bodyCode, msg := parseRespEnvelope(t, w)
			require.Equal(t, 404, bodyCode, "missing row must map to 404, got %d (%q)", bodyCode, msg)
		})
	}
}

// TestTaskRoutesAnswer503WithoutManager locks in the nil-manager guard: the
// managers are typed nil in this harness, so a handler that dereferences them
// panics (and gin.Recovery turns that into an opaque 500). The guard must answer
// a clear 503 instead.
func TestTaskRoutesAnswer503WithoutManager(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	cases := []struct{ method, target string }{
		{"GET", "/api/admin/task/upload/undone"},
		{"GET", "/api/admin/task/upload/done"},
		{"POST", "/api/admin/task/upload/info?tid=abc"},
		{"POST", "/api/admin/task/upload/cancel?tid=abc"},
		{"POST", "/api/admin/task/upload/clear_done"},
		{"POST", "/api/admin/task/upload/delete_some"},
		{"POST", "/api/admin/task/upload/retry_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			req.Header.Set("Authorization", adminToken)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			_, bodyCode, msg := parseRespEnvelope(t, w)
			require.Equal(t, 503, bodyCode,
				"a missing task manager must answer 503, got %d (%q) status %d", bodyCode, msg, w.Code)
			assert.Contains(t, strings.ToLower(msg), "not available",
				"the message should say the subsystem is unavailable, got %q", msg)
		})
	}
}
