package suspend_manager

import (
    "context"
    "errors"
    "log"
    "os"
    "os/exec"
    "time"

    "github.com/godbus/dbus/v5"
    pmpb "chromiumos/system_api/power_manager_proto"
    "jemaos.com/power_daemon/dbusutil"
)

const (
    // D-Bus signal names for suspend and resume events.
    sigSuspendImminent = "SuspendImminent"
    sigSuspendDone     = "SuspendDone"

    // D-Bus method names for suspend delay handling.
    methdRegisterSuspendDelay    = "RegisterSuspendDelay"
    methdUnregisterSuspendDelay  = "UnregisterSuspendDelay"
    methdHandleSuspendReadiness  = "HandleSuspendReadiness"

    // Paths to pre-suspend and post-resume scripts.
    pathPreSuspendScript = "/etc/powerd/pre_suspend.sh"
    pathPostResumeScript = "/etc/powerd/post_resume.sh"

    // Description of the suspend manager.
    serverDescription = "JemaOS Suspend Manager"

    // Timeout for script execution in milliseconds.
    execTimeout = 200
)

// SuspendManager manages suspend and resume events, including executing scripts
// and interacting with the D-Bus Power Manager service.
type SuspendManager struct {
    ctx             context.Context
    obj             dbus.BusObject
    delay_id        int32
    suspend_id      int32
    on_suspend_delay bool
}

// NewSuspendManager initializes a new SuspendManager instance.
func NewSuspendManager(ctx context.Context, conn *dbus.Conn) *SuspendManager {
    return &SuspendManager{ctx, dbusutil.GetPMObject(conn), 0, 0, false}
}

// sendSuspendReadiness notifies the Power Manager that the system is ready to suspend.
func (manager *SuspendManager) sendSuspendReadiness() error {
    req := &pmpb.SuspendReadinessInfo{DelayId: &manager.delay_id, SuspendId: &manager.suspend_id}
    return dbusutil.CallProtoMethod(manager.ctx, manager.obj, dbusutil.GetPMMethod(methdHandleSuspendReadiness), req, nil)
}

// handleSuspend processes the SuspendImminent signal and executes the pre-suspend script.
func (manager *SuspendManager) handleSuspend(signal *dbus.Signal) error {
    log.Println("Received Suspend signal")
    if manager.on_suspend_delay {
        return errors.New("system is already in suspend state")
    }

    suspendInfo := &pmpb.SuspendImminent{}
    if err := dbusutil.DecodeSignal(signal, suspendInfo); err != nil {
        return err
    }

    manager.suspend_id = suspendInfo.GetSuspendId()
    manager.on_suspend_delay = true
    log.Printf("On suspend: %d, reason: %s", manager.suspend_id, suspendInfo.GetReason().String())

    if _, err := os.Stat(pathPreSuspendScript); err != nil {
        log.Printf("The script %s does not exist.", pathPreSuspendScript)
    }

    ctx, cancel := context.WithTimeout(context.Background(), execTimeout*time.Millisecond)
    defer cancel()
    defer manager.sendSuspendReadiness()

    if err := exec.CommandContext(ctx, pathPreSuspendScript).Run(); err != nil {
        log.Printf("Error executing pre-suspend script: %v", err)
    }
    return nil
}

// handleResume processes the SuspendDone signal and executes the post-resume script.
func (manager *SuspendManager) handleResume(signal *dbus.Signal) error {
    log.Println("Received Resume signal")
    if !manager.on_suspend_delay {
        return errors.New("system is not in suspend state")
    }

    suspendInfo := &pmpb.SuspendDone{}
    if err := dbusutil.DecodeSignal(signal, suspendInfo); err != nil {
        return err
    }

    if suspendInfo.GetSuspendId() != manager.suspend_id {
        log.Println("The resume suspend ID is different from the original")
    }

    manager.suspend_id = 0
    manager.on_suspend_delay = false
    log.Printf("Resume complete: duration: %d, wakeup type: %s", suspendInfo.GetSuspendDuration(), suspendInfo.GetWakeupType().String())

    if _, err := os.Stat(pathPostResumeScript); err != nil {
        log.Printf("The script %s does not exist.", pathPostResumeScript)
    }

    ctx, cancel := context.WithTimeout(context.Background(), execTimeout*time.Millisecond)
    defer cancel()

    if err := exec.CommandContext(ctx, pathPostResumeScript).Run(); err != nil {
        log.Printf("Error executing post-resume script: %v", err)
    }
    return nil
}

// Register registers the suspend manager with the D-Bus signal server and sets up handlers.
func (manager *SuspendManager) Register(sigServer *dbusutil.SignalServer) error {
    timeout := int64(execTimeout)
    description := serverDescription
    req := &pmpb.RegisterSuspendDelayRequest{Timeout: &timeout, Description: &description}
    rsp := &pmpb.RegisterSuspendDelayReply{}

    if err := dbusutil.CallProtoMethod(manager.ctx, manager.obj, dbusutil.GetPMMethod(methdRegisterSuspendDelay), req, rsp); err != nil {
        return err
    }

    manager.delay_id = rsp.GetDelayId()

    suspendHandler := func(sig *dbus.Signal) error {
        return manager.handleSuspend(sig)
    }
    resumeHandler := func(sig *dbus.Signal) error {
        return manager.handleResume(sig)
    }

    sigServer.RegisterSignalHandler(sigSuspendImminent, suspendHandler)
    sigServer.RegisterSignalHandler(sigSuspendDone, resumeHandler)

    log.Println("Suspend manager registered")
    return nil
}

// UnRegister unregisters the suspend manager from the D-Bus signal server.
func (manager *SuspendManager) UnRegister(sigServer *dbusutil.SignalServer) error {
    if manager.delay_id != 0 {
        req := &pmpb.UnregisterSuspendDelayRequest{DelayId: &manager.delay_id}
        log.Println("Unregistering suspend manager")
        return dbusutil.CallProtoMethod(manager.ctx, manager.obj, dbusutil.GetPMMethod(methdUnregisterSuspendDelay), req, nil)
    }
    return nil
}