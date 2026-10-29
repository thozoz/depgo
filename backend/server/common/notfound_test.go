package common

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"gorm record not found", gorm.ErrRecordNotFound, true},
		{"wrapped gorm", errors.Join(errors.New("ctx"), gorm.ErrRecordNotFound), true},
		{"custom record not found", errors.New("failed get old meta: record not found"), true},
		{"wrapped db get", errors.New("failed get role: record not found"), true},
		{"unrelated failure", errors.New("connection refused"), false},
		{"permission failure", errors.New("you have no permission"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsNotFound(tc.err))
		})
	}
}

func TestGetErrRespMapsMissingRowTo404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"missing row", gorm.ErrRecordNotFound, 404},
		{"wrapped missing row", errors.New("failed get user: record not found"), 404},
		{"server fault", errors.New("disk exploded"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			GetErrResp(c, tc.err)
			require.Equal(t, 200, w.Code, "the wire status stays 200; the code lives in the body")

			var resp struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.want, resp.Code, "message=%q", resp.Message)
		})
	}
}
