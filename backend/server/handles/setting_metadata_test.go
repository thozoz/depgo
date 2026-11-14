package handles_test

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/server/handles"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenUpdatesPreserveSettingMetadata(t *testing.T) {
	for _, route := range []string{"set_token", "reset_token"} {
		t.Run(route, func(t *testing.T) {
			setupTestServer(t)
			// These legacy handlers are intentionally not registered in depgo.
			r := gin.New()
			r.POST("/set_token", handles.SetToken)
			r.POST("/reset_token", handles.ResetToken)
			item, err := db.GetSettingItemByKey(conf.Token)
			require.NoError(t, err)
			item.Index = 123
			item.Help = "preserve token metadata"
			item.Type = "string"
			item.Options = "keep options"
			item.Flag = model.PRIVATE
			require.NoError(t, db.SaveSettingItem(item))
			op.SettingCacheUpdate()

			req := httptest.NewRequest("POST", "/"+route, strings.NewReader(`{"token":"rotated-test-token"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, 200, parseRespCode(t, w), w.Body.String())
			stored, err := db.GetSettingItemByKey(conf.Token)
			require.NoError(t, err)
			assert.NotEqual(t, item.Value, stored.Value)
			if route == "set_token" {
				assert.Equal(t, "rotated-test-token", stored.Value)
			}
			stored.Value = item.Value
			assert.Equal(t, *item, *stored)
		})
	}
}

func TestSettingValueUpdatesDoNotMutateCachedRows(t *testing.T) {
	setupTestServer(t)
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			key := "snapshot-" + t.Name()
			original := model.SettingItem{Key: key, Value: "old", Index: 42, Type: "string", Help: "keep", Options: "options", Flag: model.PRIVATE}
			require.NoError(t, db.SaveSettingItem(&original))
			cached, err := op.GetSettingItemByKey(key)
			require.NoError(t, err)
			if batch {
				require.NoError(t, op.SetSettingValues(map[string]string{key: "new"}))
			} else {
				require.NoError(t, op.SetSettingValue(key, "new"))
			}
			assert.Equal(t, original, *cached, "published cached rows must remain immutable")
			stored, err := db.GetSettingItemByKey(key)
			require.NoError(t, err)
			assert.Equal(t, "new", stored.Value)
			stored.Value = original.Value
			assert.Equal(t, original, *stored)
		})
	}
}
