package dbusutil

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/godbus/dbus/v5"
)

// SignalHandler defines a function type for handling D-Bus signals.
type SignalHandler func(*dbus.Signal) error

// SignalHandlers is a slice of SignalHandler functions.
type SignalHandlers []SignalHandler

// SignalMap maps signal names to their respective handlers.
type SignalMap map[string]*SignalHandlers

// SignalServer manages D-Bus signal registration and handling.
type SignalServer struct {
    ctx    context.Context
    conn   *dbus.Conn
    sigmap SignalMap
}

// NewSignalServer initializes a new SignalServer instance.
func NewSignalServer(ctx context.Context, conn *dbus.Conn) *SignalServer {
    return &SignalServer{ctx, conn, make(SignalMap)}
}

// RegisterSignalHandler registers a handler for a specific D-Bus signal.
func (sigServer *SignalServer) RegisterSignalHandler(sigName string, handler SignalHandler) {
    handlers, ok := sigServer.sigmap[sigName]
    if !ok {
        buff := make(SignalHandlers, 0, 5)
        sigServer.sigmap[sigName] = &buff
        handlers = sigServer.sigmap[sigName]
    }
    *handlers = append(*handlers, handler)
}

// addMatchSignal adds a match rule for a specific D-Bus signal.
func (sigServer *SignalServer) addMatchSignal(sigName string) error {
    log.Printf("Add signal filter path:%s, interface:%s, signal:%s",
        PowerManagerPath, PowerManagerInterface, sigName)
    return sigServer.conn.AddMatchSignal(
        dbus.WithMatchObjectPath(PowerManagerPath),
        dbus.WithMatchInterface(PowerManagerInterface),
        dbus.WithMatchMember(sigName),
    )
}

// removeMatchSignal removes a match rule for a specific D-Bus signal.
func (sigServer *SignalServer) removeMatchSignal(sigName string) error {
    log.Printf("Remove signal filter path:%s, interface:%s, signal:%s",
        PowerManagerPath, PowerManagerInterface, sigName)
    return sigServer.conn.RemoveMatchSignal(
        dbus.WithMatchObjectPath(PowerManagerPath),
        dbus.WithMatchInterface(PowerManagerInterface),
        dbus.WithMatchMember(sigName),
    )
}

// addAllSignals adds match rules for all registered signals.
func (sigServer *SignalServer) addAllSignals() {
    for name := range sigServer.sigmap {
        if err := sigServer.addMatchSignal(name); err != nil {
            log.Printf("Add signal %s, got error: %v", name, err)
        }
    }
    log.Println("Finished adding signal filters.")
}

// removeAllSignals removes match rules for all registered signals.
func (sigServer *SignalServer) removeAllSignals() {
    for name := range sigServer.sigmap {
        if err := sigServer.removeMatchSignal(name); err != nil {
            log.Printf("Remove signal %s, got error: %v", name, err)
        }
    }
    sigServer.sigmap = nil
}

// handleSignal processes an incoming D-Bus signal and invokes its handlers.
func (sigServer *SignalServer) handleSignal(sig *dbus.Signal) {
    member := sig.Name[len(PowerManagerInterface)+1:]
    log.Printf("Received Signal %s, member: %s", sig.Name, member)
    if handlers, ok := sigServer.sigmap[member]; ok {
        for _, h := range *handlers {
            if h != nil {
                if err := h(sig); err != nil {
                    log.Printf("Handler signal error: %v", err)
                }
            }
        }
    }
}

// StartWorking starts the signal server to listen for D-Bus signals.
func (sigServer *SignalServer) StartWorking() {
    sigServer.addAllSignals()
    defer sigServer.removeAllSignals()

    ch := make(chan *dbus.Signal, 10)
    defer close(ch)
    sigServer.conn.Signal(ch)
    defer sigServer.conn.RemoveSignal(ch)

    log.Println("Start listening for signals...")
    sysch := make(chan os.Signal, 1)
    signal.Notify(sysch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGABRT)

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