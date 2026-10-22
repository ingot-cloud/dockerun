package builtin

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestMaintenanceLeasePreservesManualSetting(t *testing.T) {
	for _, manual := range []bool{false, true} {
		if err := SetReadOnly(manual); err != nil {
			t.Fatal(err)
		}
		finish, err := BeginMaintenance(context.Background(), "backup")
		if err != nil {
			t.Fatal(err)
		}
		state := MaintenanceState()
		if !state.ReadOnly || state.ManualReadOnly != manual || state.Operation != "backup" {
			t.Fatalf("unexpected active state: %+v", state)
		}
		if err := SetReadOnly(!manual); !errors.Is(err, ErrMaintenanceBusy) {
			t.Errorf("write state changed during backup: %v", err)
		}
		if _, err := BeginMaintenance(context.Background(), "registry_gc"); !errors.Is(err, ErrMaintenanceBusy) {
			t.Errorf("concurrent maintenance accepted: %v", err)
		}
		gate.Lock()
		stateRead := make(chan MaintenanceStatus, 1)
		go func() { stateRead <- MaintenanceState() }()
		select {
		case <-stateRead:
		case <-time.After(time.Second):
			t.Error("status blocked on the storage gate")
		}
		gate.Unlock()
		finish()
		finish()
		state = MaintenanceState()
		if state.ReadOnly != manual || state.ManualReadOnly != manual || state.Operation != "" {
			t.Fatalf("manual state was overwritten: %+v", state)
		}
	}
	SetReadOnly(false)
}
func TestMaintenanceCancellationWhileRequestsDrain(t *testing.T) {
	SetReadOnly(false)
	gate.RLock()
	defer gate.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := BeginMaintenance(ctx, "backup"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("maintenance ignored cancellation: %v", err)
	}
	if state := MaintenanceState(); state.ReadOnly || state.Operation != "" {
		t.Fatalf("cancelled maintenance leaked state: %+v", state)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := SetReadOnlyContext(ctx, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("toggle ignored cancellation: %v", err)
	}
	if ReadOnly() {
		t.Fatal("failed toggle did not restore the previous setting")
	}
}
func TestGarbageCollectionRequiresAnEnabledManualWindow(t *testing.T) {
	previous := Engine
	defer func() { Engine = previous; SetReadOnly(false) }()
	SetReadOnly(false)
	Engine = nil
	if _, err := BeginMaintenance(context.Background(), "registry_gc"); err == nil {
		t.Fatal("disabled registry accepted GC")
	}
	Engine = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if _, err := BeginMaintenance(context.Background(), "registry_gc"); err == nil {
		t.Fatal("writable registry accepted GC")
	}
	if err := SetReadOnly(true); err != nil {
		t.Fatal(err)
	}
	finish, err := BeginMaintenance(context.Background(), "registry_gc")
	if err != nil {
		t.Fatal(err)
	}
	finish()
	if !ReadOnly() {
		t.Fatal("GC ended the administrator's maintenance window")
	}
}
