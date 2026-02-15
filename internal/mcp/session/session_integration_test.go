//go:build integration

package session

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

func setupValkey(t *testing.T) valkey.Client {
	t.Helper()
	addr := os.Getenv("TEST_VALKEY_ADDR")
	if addr == "" {
		t.Fatal("TEST_VALKEY_ADDR not set")
	}
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		t.Skipf("valkey not available: %v", err)
	}
	// Verify connectivity
	ctx := context.Background()
	resp := client.Do(ctx, client.B().Ping().Build())
	if resp.Error() != nil {
		t.Skipf("valkey ping failed: %v", resp.Error())
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestManager_SaveAndLoad(t *testing.T) {
	client := setupValkey(t)
	mgr := NewManager(client)
	ctx := context.Background()

	// Create and populate a session
	sess, err := mgr.Load(ctx, "")
	if err != nil {
		t.Fatalf("load new session: %v", err)
	}
	if sess.ID == "" {
		t.Fatal("auto-generated ID should not be empty")
	}

	id1, id2 := uuid.New(), uuid.New()
	sess.MarkSeen(id1, id2)
	sess.AddQuery("search for customers")
	sess.UpdateFocus(id1)
	sess.AddWaypoint(id2, "key table")
	sess.AddRecap("Found CustomerRepository")

	if err := mgr.Save(ctx, sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	// Reload and verify
	loaded, err := mgr.Load(ctx, sess.ID)
	if err != nil {
		t.Fatalf("reload session: %v", err)
	}

	if loaded.ID != sess.ID {
		t.Errorf("ID mismatch: %q vs %q", loaded.ID, sess.ID)
	}
	if !loaded.IsSeen(id1) {
		t.Error("id1 should be seen after reload")
	}
	if !loaded.IsSeen(id2) {
		t.Error("id2 should be seen after reload")
	}
	if len(loaded.QueryHistory) != 1 || loaded.QueryHistory[0] != "search for customers" {
		t.Errorf("query history not preserved: %v", loaded.QueryHistory)
	}
	if len(loaded.FocusArea) != 1 {
		t.Errorf("focus area not preserved: %v", loaded.FocusArea)
	}
	if len(loaded.Waypoints) != 1 || loaded.Waypoints[0].Label != "key table" {
		t.Error("waypoints not preserved")
	}
	if len(loaded.Recap) != 1 {
		t.Error("recap not preserved")
	}

	// Cleanup
	client.Do(ctx, client.B().Del().Key(sessionKeyPrefix+sess.ID).Build())
}

func TestManager_Load_NonexistentSession(t *testing.T) {
	client := setupValkey(t)
	mgr := NewManager(client)
	ctx := context.Background()

	sess, err := mgr.Load(ctx, "nonexistent-session-"+uuid.New().String())
	if err != nil {
		t.Fatalf("load nonexistent should not error: %v", err)
	}
	if sess.SeenSymbols == nil {
		t.Error("new session should have initialized SeenSymbols")
	}
}

func TestManager_Load_EmptyID_GeneratesNew(t *testing.T) {
	client := setupValkey(t)
	mgr := NewManager(client)
	ctx := context.Background()

	sess, err := mgr.Load(ctx, "")
	if err != nil {
		t.Fatalf("load empty ID: %v", err)
	}
	if sess.ID == "" {
		t.Error("should auto-generate an ID")
	}
	// Verify it's a valid UUID
	if _, err := uuid.Parse(sess.ID); err != nil {
		t.Errorf("auto-generated ID should be valid UUID: %v", err)
	}
}

func TestManager_UpdatedAt_Changes(t *testing.T) {
	client := setupValkey(t)
	mgr := NewManager(client)
	ctx := context.Background()

	sess, _ := mgr.Load(ctx, "")
	originalUpdated := sess.UpdatedAt

	time.Sleep(10 * time.Millisecond)
	if err := mgr.Save(ctx, sess); err != nil {
		t.Fatalf("save: %v", err)
	}

	if !sess.UpdatedAt.After(originalUpdated) {
		t.Error("UpdatedAt should advance on save")
	}

	// Cleanup
	client.Do(ctx, client.B().Del().Key(sessionKeyPrefix+sess.ID).Build())
}

func TestManager_SessionIsolation(t *testing.T) {
	client := setupValkey(t)
	mgr := NewManager(client)
	ctx := context.Background()

	sess1, _ := mgr.Load(ctx, "")
	sess2, _ := mgr.Load(ctx, "")

	id := uuid.New()
	sess1.MarkSeen(id)
	mgr.Save(ctx, sess1)

	// Reload sess2 — should NOT see sess1's symbols
	loaded2, _ := mgr.Load(ctx, sess2.ID)
	if loaded2.IsSeen(id) {
		t.Error("sessions should be isolated — sess2 should not see sess1's symbols")
	}

	// Cleanup
	client.Do(ctx, client.B().Del().Key(sessionKeyPrefix+sess1.ID).Build())
	client.Do(ctx, client.B().Del().Key(sessionKeyPrefix+sess2.ID).Build())
}
