package share

import (
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/model"
)

func TestTokenPayloadExcludesPasswordHash(t *testing.T) {
	const hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	share := &model.Share{
		ShareID:        "abc123",
		PasswordHash:   hash,
		PasswordSalt:   "somesalt",
		AccessRevision: 7,
	}

	payload := tokenPayload(share)
	if strings.Contains(payload, hash) {
		t.Fatalf("token payload leaks the password hash: %q", payload)
	}
	if strings.Contains(payload, share.PasswordSalt) {
		t.Fatalf("token payload leaks the password salt: %q", payload)
	}
	if want := "abc123:7"; payload != want {
		t.Fatalf("payload = %q, want %q", payload, want)
	}
}

func TestTokenPayloadChangesWithRevision(t *testing.T) {
	share := &model.Share{ShareID: "abc123", AccessRevision: 1}
	before := tokenPayload(share)
	share.AccessRevision = 2
	after := tokenPayload(share)
	if before == after {
		t.Fatalf("payload must change when AccessRevision changes (%q)", before)
	}
}

func TestTokenPayloadIgnoresPasswordHash(t *testing.T) {
	share := &model.Share{ShareID: "abc123", PasswordHash: "hash-a", AccessRevision: 1}
	first := tokenPayload(share)
	// A password change increments AccessRevision on the handler side, which is
	// what invalidates tokens; the hash itself must not be part of the payload.
	share.PasswordHash = "hash-b"
	if tokenPayload(share) != first {
		t.Fatalf("payload must not depend on PasswordHash: %q", first)
	}
}

func TestTokenPayloadIsShareScoped(t *testing.T) {
	a := tokenPayload(&model.Share{ShareID: "share-a", AccessRevision: 1})
	b := tokenPayload(&model.Share{ShareID: "share-b", AccessRevision: 1})
	if a == b {
		t.Fatalf("different shares must produce different payloads (%q)", a)
	}
}
