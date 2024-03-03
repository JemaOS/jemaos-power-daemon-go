package suspend_manager

import (
  "context"
  "os"
  "os/exec"
  "time"
  "log"
  "errors"
  "github.com/godbus/dbus/v5"
  pmpb "chromiumos/system_api/power_manager_proto"
  "jemaos.com/power_daemon/dbusutil"
)

const (
  sigSuspendImminent = "SuspendImminent"
  sigSuspendDone = "SuspendDone"
  methdRegisterSuspendDelay = "RegisterSuspendDelay"
  methdUnregisterSuspendDelay = "UnregisterSuspendDelay"
  methdHandleSuspendReadiness = "HandleSuspendReadiness"
  pathPreSuspendScript = "/etc/powerd/pre_suspend.sh"
  pathPostResumeScript = "/etc/powerd/post_resume.sh"
  serverDescription = "JemaOS Suspend Manager"
  execTimeout = 200
)

type SuspendManager struct {
  ctx context.Context
  obj dbus.BusObject
  delay_id int32
  suspend_id int32
  on_suspend_delay bool
}

func NewSuspendManager(ctx context.Context, conn *dbus.Conn) *SuspendManager {
  return &SuspendManager{ctx, dbusutil.GetPMObject(conn), 0, 0, false}
}

func (manager *SuspendManager) sendSuspendReadiness() error{
  req := &pmpb.SuspendReadinessInfo{ DelayId: &manager.delay_id, SuspendId: &manager.suspend_id}
  return dbusutil.CallProtoMethod(manager.ctx, manager.obj, dbusutil.GetPMMethod(methdHandleSuspendReadiness), req, nil)
}

func (manager *SuspendManager) handleSuspend(signal *dbus.Signal) error {
  log.Println("Get Suspend signal")
  if manager.on_suspend_delay {
    return errors.New("System is on suspend already")
  }
  suspendInfo := &pmpb.SuspendImminent{}
  if err := dbusutil.DecodeSignal(signal, suspendInfo); err != nil {
    return err
  }
  manager.suspend_id = suspendInfo.GetSuspendId()
  manager.on_suspend_delay = true
  log.Printf("On suspend: %d, for reason %s", manager.suspend_id, suspendInfo.GetReason().String())
  if _, err := os.Stat(pathPreSuspendScript); err != nil {
    log.Printf("The script:%s is not exist.", pathPreSuspendScript)
  }
  ctx, cancel := context.WithTimeout(context.Background(), execTimeout * time.Millisecond)
  defer cancel()
  defer manager.sendSuspendReadiness()
  if err := exec.CommandContext(ctx, pathPreSuspendScript).Run(); err != nil {
    log.Printf("Exec pre-suspend script error:%w", err)
  }
  return nil
}

func (manager *SuspendManager) handleResume(signal *dbus.Signal) error {
  log.Println("Get Resume signal")
  if !manager.on_suspend_delay {
    return errors.New("System is not on suspend")
  }
  suspendInfo := &pmpb.SuspendDone{}
  if err := dbusutil.DecodeSignal(signal, suspendInfo); err != nil {
    return err
  }
  if suspendInfo.GetSuspendId() != manager.suspend_id {
    log.Println("The resume suspend id is different from original")
  }
  manager.suspend_id = 0
  manager.on_suspend_delay = false
  log.Printf("On suspend: %d, duration: %d, type:%s", manager.suspend_id, suspendInfo.GetSuspendDuration(), suspendInfo.GetWakeupType().String())
  if _, err := os.Stat(pathPreSuspendScript); err != nil {
    log.Printf("The script:%s is not exist.", pathPreSuspendScript)
  }
  ctx, cancel := context.WithTimeout(context.Background(), execTimeout * time.Millisecond * 1000)
  defer cancel()
  if err := exec.CommandContext(ctx, pathPostResumeScript).Run(); err != nil {
    log.Printf("Exec post-resume script error:%w", err)
  }
  return nil
}

func (manager *SuspendManager) Register(sigServer *dbusutil.SignalServer) error {
  var suspend_handler,resume_handler dbusutil.SignalHandler
  timeout := int64(execTimeout)
  descript:= serverDescription
  req := &pmpb.RegisterSuspendDelayRequest{Timeout: &timeout, Description: &descript}
  rsp := &pmpb.RegisterSuspendDelayReply{}
  err := dbusutil.CallProtoMethod(manager.ctx, manager.obj, dbusutil.GetPMMethod(methdRegisterSuspendDelay), req, rsp)
  if err != nil {
    return err
  }
  manager.delay_id = rsp.GetDelayId();
  suspend_handler = func(sig *dbus.Signal) error{
        return manager.handleSuspend(sig)}
  sigServer.RegisterSignalHandler(sigSuspendImminent, suspend_handler)
  resume_handler = func(sig *dbus.Signal) error {
        return manager.handleResume(sig)}
  sigServer.RegisterSignalHandler(sigSuspendDone, resume_handler)
  log.Println("Register suspend manager")
  return nil
}

func (manager *SuspendManager) UnRegister(sigServer *dbusutil.SignalServer) error {
  if manager.delay_id != 0 {
    req := &pmpb.UnregisterSuspendDelayRequest{DelayId: &manager.delay_id}
    log.Println("Unregister suspend manager")
    return dbusutil.CallProtoMethod(manager.ctx, manager.obj, dbusutil.GetPMMethod(methdUnregisterSuspendDelay), req, nil)
  }
  return nil
}
