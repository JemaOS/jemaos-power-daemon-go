package main

import (
  "context"
  "time"
  "log"
  "os"
  "github.com/godbus/dbus/v5"
  "jemaos.com/power_daemon/dbusutil"
  "jemaos.com/power_daemon/suspend_manager"
  "jemaos.com/power_daemon/backlight_manager"
)


func main() {
  log.SetOutput(os.Stdout)
  log.Println("Waiting for power manager init...")
  time.Sleep(1000 * time.Millisecond)
  log.Println("Trying connect system bus")
  conn, err := dbus.ConnectSystemBus(dbus.WithSignalHandler(dbus.NewSequentialSignalHandler()))
  if err != nil {
    log.Fatalf("Connect system bus error:%w", err)
  }
  defer conn.Close()
  ctx,cancel := context.WithCancel(context.Background())
  defer cancel()
  sigServer := dbusutil.NewSignalServer(ctx, conn)
  suspendManager := suspend_manager.NewSuspendManager(ctx, conn)
  if err := suspendManager.Register(sigServer); err != nil {
    log.Fatalf("suspend manager register error:%w", err)
  }
  defer suspendManager.UnRegister(sigServer)
  backlightManager := backlight_manager.NewScreenBrightnessManager(ctx, conn)
  if err := backlightManager.Register(sigServer); err != nil {
    log.Fatalf("backlight manager register error:%w", err)
  }
  defer backlightManager.UnRegister(sigServer)
  sigServer.StartWorking()
}
