package backlight_manager

import (
  "context"
  "log"
  "os"
  "os/exec"
  "time"
  "fmt"
  "github.com/godbus/dbus/v5"
  "strconv"
  "io/ioutil"
  pmpb "chromiumos/system_api/power_manager_proto"
  "jemaos.com/power_daemon/dbusutil"
)

const (
  sigScreenBrightnessChanged = "ScreenBrightnessChanged"
  sigKeyBoardBrightnessChanged = "KeyboardBrightnessChanged"
  methdSetScreenBrightness = "SetScreenBrightness"
  pathConfig = "/mnt/stateful_partition/unencrypted/hwconfig"
  fileBrightness = "ScreenBrightness"
  fileKeyboardBrightness = "KeyBoardBrightness"
  defaultBrightness = 60.0
  minBrightness = 10.0
  backlightTool = "/usr/bin/backlight_tool"
)

type ScreenBrightnessManager struct {
  ctx context.Context
  obj dbus.BusObject
  screen_brightness float64
  need_store_screen bool
  keyboard_brightness float64
  need_store_keyboard bool
}

func getHWConfig(name string) (string, error) {
  log.Printf("get config:%s", name);
  fi, err := os.Lstat(pathConfig)
  if err != nil || !fi.IsDir() {
    return "",fmt.Errorf("%s or %s is not exist", pathConfig, name)
  }
  buf, err := ioutil.ReadFile(pathConfig + "/" + name)
  if err != nil {
    return "", err
  }
  log.Printf("get config:%s, value:%s", name, buf)
  return string(buf), nil
}

func saveHWConfig(name string, value string) error {
  log.Printf("save config %s:%s", name, value)
  _, err := os.Lstat(pathConfig)
  if err != nil {
    err = os.Mkdir(pathConfig, 0700)
    if err != nil {
      return err
    }
  }
  return ioutil.WriteFile(pathConfig + "/" + name, []byte(value), 0644)
}

func NewScreenBrightnessManager(ctx context.Context, conn *dbus.Conn) (bm *ScreenBrightnessManager) {
  bm = &ScreenBrightnessManager{ctx, dbusutil.GetPMObject(conn),
      defaultBrightness, false, 0, false}
  if value, err := getHWConfig(fileBrightness); err == nil {
    log.Printf("read hardware config; screen brightness:%s", value)
    bm.screen_brightness, _ = strconv.ParseFloat(value, 64)
  }else {
    log.Printf("read error:", err)
  }
  if value, err := getHWConfig(fileKeyboardBrightness); err == nil {
    log.Printf("read hardware config; keyboard brightness:%s", value)
    bm.keyboard_brightness, _ = strconv.ParseFloat(value, 64)
  }else {
    log.Printf("read error:", err)
  }
  return
}

func (bm *ScreenBrightnessManager) HandleSetScreenBrightness(signal *dbus.Signal) error {
  log.Println("Get Set Screen Brightness signal")
  brightChg := &pmpb.BacklightBrightnessChange{}
  if err := dbusutil.DecodeSignal(signal, brightChg); err != nil {
    return err
  }
  if (brightChg.GetCause() == pmpb.BacklightBrightnessChange_USER_REQUEST) {
    if brightChg.GetPercent() > minBrightness && bm.screen_brightness != brightChg.GetPercent() {
      bm.screen_brightness = brightChg.GetPercent()
      bm.need_store_screen = true
    }
    log.Printf("User set screen brightness to %v", bm.screen_brightness)
  }
  return nil
}

func (bm *ScreenBrightnessManager) HandleSetKeyboardBrightness(signal *dbus.Signal) error {
  log.Println("Get Set Keyboard Brightness signal")
  brightChg := &pmpb.BacklightBrightnessChange{}
  if err := dbusutil.DecodeSignal(signal, brightChg); err != nil {
    return err
  }
  if (brightChg.GetCause() == pmpb.BacklightBrightnessChange_USER_REQUEST) {
    if bm.keyboard_brightness != brightChg.GetPercent() {
      bm.keyboard_brightness = brightChg.GetPercent()
      bm.need_store_keyboard = true
    }
    log.Printf("User set keyboard brightness to %v", bm.keyboard_brightness)
  }
  return nil
}

func (bm *ScreenBrightnessManager) SetScreenBrightness() error {
  log.Printf("Set screen brightness to: %v", bm.screen_brightness)
  trans := pmpb.SetBacklightBrightnessRequest_INSTANT
  cause := pmpb.SetBacklightBrightnessRequest_MODEL
  req := &pmpb.SetBacklightBrightnessRequest{
    Percent: &bm.screen_brightness,
    Transition: &trans,
    Cause: &cause,
  }
  return dbusutil.CallProtoMethod(bm.ctx, bm.obj, dbusutil.GetPMMethod(methdSetScreenBrightness), req, nil)
}

func (bm *ScreenBrightnessManager) SetKeyboardBrightness() error {
  ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
  defer cancel()
  log.Printf("Set keyboard backlight to: %v", bm.keyboard_brightness)
  brightnessArg := fmt.Sprintf("--set_brightness_percent=%.1f", bm.keyboard_brightness)
  return exec.CommandContext(ctx, backlightTool, "--keyboard", brightnessArg).Run()
}

func (bm *ScreenBrightnessManager) Register(sigServer *dbusutil.SignalServer) error {
  if err := bm.SetScreenBrightness(); err != nil {
   log.Printf("Set screen brightness error:%w", err)
  }
  if err := bm.SetKeyboardBrightness(); err != nil {
   log.Printf("Set keyboard brightness error:%w", err)
  }
  var sbl_handler, kbl_handler dbusutil.SignalHandler
  sbl_handler = func(sig *dbus.Signal) error {return bm.HandleSetScreenBrightness(sig)}
  kbl_handler = func(sig *dbus.Signal) error {return bm.HandleSetKeyboardBrightness(sig)}
  sigServer.RegisterSignalHandler(sigScreenBrightnessChanged, sbl_handler)
  sigServer.RegisterSignalHandler(sigKeyBoardBrightnessChanged, kbl_handler)
  log.Println("Register brightness manager")
  return nil
}

func (bm *ScreenBrightnessManager) UnRegister(sigServer *dbusutil.SignalServer) error {
  if bm.need_store_screen {
    if err := saveHWConfig(fileBrightness, strconv.FormatFloat(bm.screen_brightness, 'f', 1, 64)); err != nil {
      log.Printf("Get error when save %s, error: %w", fileBrightness, err)
    }
  }
  if bm.need_store_keyboard {
    if err := saveHWConfig(fileKeyboardBrightness, strconv.FormatFloat(bm.keyboard_brightness, 'f', 1, 64)); err != nil {
      log.Printf("Get error when save %s, error: %w", fileKeyboardBrightness, err)
    }
  }
  log.Println("Unregister brightness manager")
  return nil
}
