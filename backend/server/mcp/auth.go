package mcp

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/device"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/server/common"
	log "github.com/sirupsen/logrus"
)

type ctxKey string

const userKey ctxKey = "user"

// HTTPContextFunc extracts JWT/admin token from HTTP request and injects user into context.
// Used as WithHTTPContextFunc callback for Streamable HTTP transport.
func HTTPContextFunc(ctx context.Context, r *http.Request) context.Context {
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	user, deviceKey, err := authenticateToken(token)
	if err != nil {
		log.Debugf("MCP auth failed: %v", err)
		return ctx
	}
	if err := requireActiveSession(user, deviceKey); err != nil {
		log.Debugf("MCP auth rejected: %v", err)
		return ctx
	}

	return context.WithValue(ctx, userKey, user)
}

// authenticateToken resolves the credential to a user and, for a login token,
// the device session it was minted for. Admin-token and guest credentials have
// no session and return an empty device key.
func authenticateToken(token string) (*model.User, string, error) {
	// Check admin static token
	if token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(setting.GetStr(conf.Token))) == 1 {
		admin, err := op.GetAdmin()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get admin: %w", err)
		}
		if err := loadRoles(admin); err != nil {
			return nil, "", err
		}
		return admin, "", nil
	}

	// No token: guest
	if token == "" {
		guest, err := op.GetGuest()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get guest: %w", err)
		}
		if guest.Disabled {
			return nil, "", fmt.Errorf("guest user is disabled")
		}
		if err := loadRoles(guest); err != nil {
			return nil, "", err
		}
		return guest, "", nil
	}

	// JWT token
	claims, err := common.ParseToken(token)
	if err != nil {
		return nil, "", fmt.Errorf("invalid token: %w", err)
	}

	user, err := op.GetUserByName(claims.Username)
	if err != nil {
		return nil, "", fmt.Errorf("user not found: %w", err)
	}

	if claims.PwdTS != user.PwdTS {
		return nil, "", fmt.Errorf("password has been changed")
	}
	if user.Disabled {
		return nil, "", fmt.Errorf("user is disabled")
	}

	if err := loadRoles(user); err != nil {
		return nil, "", err
	}
	if claims.DeviceKey == "" {
		// Tokens issued before the claim existed are refused rather than silently
		// downgraded to a header-derived session key.
		return nil, "", fmt.Errorf("token is not bound to a device session")
	}
	return user, claims.DeviceKey, nil
}

// requireActiveSession rejects a credential whose device session was retired.
//
// Each HTTP request runs middlewares.HandleSession, which notices an inactive
// session and refuses. MCP requests skipped that path entirely, so a session that
// an admin (or the max_devices policy) had evicted kept working over MCP: the
// eviction was silently ineffective on that protocol.
//
// The check is deliberately advisory rather than authoritative: if no session row
// exists for the credential, MCP still allows it. MCP clients are not browsers and
// are not required to have logged in over HTTP, so demanding an existing row would
// lock out legitimate automation. What it does catch is the case that matters -
// a session that exists and has been explicitly deactivated.
func requireActiveSession(user *model.User, deviceKey string) error {
	// Only a login token names a session. The site admin token and the guest
	// credential carry no session at all, so there is nothing to check.
	if deviceKey == "" {
		return nil
	}
	err := device.CheckActive(user.ID, deviceKey)
	if err == nil {
		return nil
	}
	if errors.Is(err, errs.SessionInactive) {
		return fmt.Errorf("session has been deactivated")
	}
	return fmt.Errorf("check device session: %w", err)
}

func loadRoles(user *model.User) error {
	if len(user.Role) > 0 {
		roles, err := op.GetRolesByUserID(user.ID)
		if err != nil {
			return fmt.Errorf("failed to load roles: %w", err)
		}
		user.RolesDetail = roles
	}
	return nil
}

// resolveUser extracts the authenticated user from context.
func resolveUser(ctx context.Context) (*model.User, error) {
	user, ok := ctx.Value(userKey).(*model.User)
	if !ok || user == nil {
		return nil, fmt.Errorf("authentication required")
	}
	return user, nil
}

// buildFsContext resolves path and sets meta in context for fs operations.
func buildFsContext(ctx context.Context, user *model.User, path string) (context.Context, string, error) {
	reqPath, err := user.JoinPath(path)
	if err != nil {
		return ctx, "", err
	}
	meta, _ := op.GetNearestMeta(reqPath)
	ctx = context.WithValue(ctx, "meta", meta)
	ctx = context.WithValue(ctx, "user", user)
	return ctx, reqPath, nil
}

// checkAccess checks if user can access the path (read).
func checkAccess(user *model.User, reqPath string) error {
	meta, _ := op.GetNearestMeta(reqPath)
	if !common.CanAccessWithRoles(user, meta, reqPath, "") {
		return fmt.Errorf("permission denied")
	}
	perm := common.MergeRolePermissions(user, reqPath)
	if !user.IsAdmin() && !common.HasPermission(perm, common.PermMCPAccess) {
		return fmt.Errorf("MCP access not permitted")
	}
	return nil
}

// checkManage checks if user can perform write operations via MCP.
func checkManage(user *model.User, reqPath string, permBit uint) error {
	if err := checkAccess(user, reqPath); err != nil {
		return err
	}
	perm := common.MergeRolePermissions(user, reqPath)
	if !user.IsAdmin() && !common.HasPermission(perm, common.PermMCPManage) {
		return fmt.Errorf("MCP manage not permitted")
	}
	if !user.IsAdmin() && !common.HasPermission(perm, permBit) {
		return fmt.Errorf("permission denied for this operation")
	}
	return nil
}

// UserContextFunc returns an HTTPContextFunc that injects a specific user (for STDIO mode).
func userContextMiddleware(user *model.User) func(ctx context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, userKey, user)
	}
}

// resolveUserForStdio resolves a user by username for STDIO mode.
func resolveUserForStdio(username string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || username == "admin" {
		admin, err := op.GetAdmin()
		if err != nil {
			return nil, fmt.Errorf("failed to get admin user: %w", err)
		}
		if err := loadRoles(admin); err != nil {
			return nil, err
		}
		return admin, nil
	}
	user, err := op.GetUserByName(username)
	if err != nil {
		return nil, fmt.Errorf("user %q not found: %w", username, err)
	}
	if user.Disabled {
		return nil, fmt.Errorf("user %q is disabled", username)
	}
	if err := loadRoles(user); err != nil {
		return nil, err
	}
	return user, nil
}
