package builtin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrMaintenanceBusy = errors.New("仓库正在执行维护操作，请等待当前任务完成")
var maintenanceMu sync.Mutex
var stateMu sync.RWMutex
var manualReadOnly bool
var activeOperation string

type MaintenanceStatus struct {
	ReadOnly       bool   `json:"read_only"`
	ManualReadOnly bool   `json:"manual_read_only"`
	Operation      string `json:"operation"`
}

// Status reads must remain available while GC holds the storage gate.
func MaintenanceState() MaintenanceStatus {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return MaintenanceStatus{manualReadOnly || activeOperation != "", manualReadOnly, activeOperation}
}
func ReadOnly() bool               { return MaintenanceState().ReadOnly }
func SetReadOnly(value bool) error { return SetReadOnlyContext(context.Background(), value) }
func SetReadOnlyContext(ctx context.Context, value bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !maintenanceMu.TryLock() {
		return ErrMaintenanceBusy
	}
	defer maintenanceMu.Unlock()
	stateMu.Lock()
	previous := manualReadOnly
	manualReadOnly = value
	stateMu.Unlock()
	// Stop new writes before waiting for in-flight requests to drain.
	if err := lockRegistry(ctx); err != nil {
		stateMu.Lock()
		manualReadOnly = previous
		stateMu.Unlock()
		return err
	}
	gate.Unlock()
	return nil
}

// BeginMaintenance enforces read-only storage without overwriting the manual
// setting. The caller must call finish on every exit path.
func BeginMaintenance(ctx context.Context, operation string) (finish func(), err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !maintenanceMu.TryLock() {
		return nil, ErrMaintenanceBusy
	}
	stateMu.Lock()
	if operation == "registry_gc" && (Engine == nil || !manualReadOnly) {
		stateMu.Unlock()
		maintenanceMu.Unlock()
		return nil, fmt.Errorf("请先启用内置仓库并开启只读维护")
	}
	activeOperation = operation
	stateMu.Unlock()
	var once sync.Once
	finish = func() {
		once.Do(func() { stateMu.Lock(); activeOperation = ""; stateMu.Unlock(); maintenanceMu.Unlock() })
	}
	if err := lockRegistry(ctx); err != nil {
		finish()
		return nil, err
	}
	gate.Unlock()
	return finish, nil
}
func lockRegistry(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if gate.TryLock() {
			return nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
