package dbusutil

import (
  "context"
  "log"
  "github.com/godbus/dbus/v5"
  "os"
  "os/signal"
  "syscall"
)

type SignalHandler func(*dbus.Signal) error

type SignalHandlers []SignalHandler

type SignalMap map[string]*SignalHandlers

type SignalServer struct {
  ctx context.Context
  conn *dbus.Conn
  sigmap SignalMap
}

func NewSignalServer(ctx context.Context, conn *dbus.Conn) *SignalServer {
  return &SignalServer{ctx, conn, make(SignalMap)}
}

func (sigServer *SignalServer) RegisterSignalHandler(sigName string, handler SignalHandler) {
  handlers, ok := sigServer.sigmap[sigName]
  if !ok {
    buff := make(SignalHandlers,0, 5)
    sigServer.sigmap[sigName] = &buff
    handlers = sigServer.sigmap[sigName]
  }
  *handlers = append(*handlers, handler)
}

func (sigServer *SignalServer) addMatchSignal(sigName string) error {
  log.Printf("Add signal filter path:%s, interface:%s, signal:%s",
    PowerManagerPath, PowerManagerInterface, sigName)
  return sigServer.conn.AddMatchSignal(dbus.WithMatchObjectPath(PowerManagerPath),
    dbus.WithMatchInterface(PowerManagerInterface),
    dbus.WithMatchMember(sigName))
}

func (sigServer *SignalServer) removeMatchSignal(sigName string) error {
  log.Printf("Remove signal filter path:%s, interface:%s, signal:%s",
    PowerManagerPath, PowerManagerInterface, sigName)
  return sigServer.conn.RemoveMatchSignal(dbus.WithMatchObjectPath(PowerManagerPath),
      dbus.WithMatchInterface(PowerManagerInterface),
          dbus.WithMatchMember(sigName))
}

func (sigServer *SignalServer) addAllSignals() {
  for name, _ := range sigServer.sigmap {
     if err := sigServer.addMatchSignal(name); err != nil {
       log.Printf("Add signal %s, got error: %w", name, err)
     }
  }
  log.Println("Finnished add signal filters.")
}

func (sigServer *SignalServer) removeAllSignals() {
  for name, _ := range sigServer.sigmap {
     if err := sigServer.removeMatchSignal(name); err != nil {
       log.Printf("Remove signal %s, got error: %w", name, err)
     }
  }
  sigServer.sigmap = nil
}

func (sigServer *SignalServer) handleSignal(sig *dbus.Signal) {
  member := sig.Name[len(PowerManagerInterface)+1:]
  log.Printf("Get Signal %s, member: %s", sig.Name, member);
  if handlers, ok := sigServer.sigmap[member]; ok {
    for _, h := range *handlers {
      if h != nil {
        if err := h(sig); err != nil {
          log.Printf("handler signal error:%w", err);
        }
      }
    }
  }
}

func (sigServer *SignalServer) StartWorking() {
  sigServer.addAllSignals()
  defer sigServer.removeAllSignals()
  ch := make(chan *dbus.Signal, 10)
  defer close(ch)
  sigServer.conn.Signal(ch)
  defer sigServer.conn.RemoveSignal(ch)
  log.Println("Start listening signal...");
  sysch := make(chan os.Signal, 1)
  signal.Notify(sysch, syscall.SIGINT, syscall.SIGQUIT,
    syscall.SIGKILL, syscall.SIGTERM, syscall.SIGABRT)
  for {
    select {
      case sig := <-ch:
        sigServer.handleSignal(sig)
      case <-sigServer.ctx.Done():
        return
      case <-sysch:
        return
    }
  }
}
