package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/godbus/dbus/v5"
    "jemaos.com/power_daemon/backlight_manager"
    "jemaos.com/power_daemon/dbusutil"
    "jemaos.com/power_daemon/suspend_manager"
)

// main is the entry point of the JemaOS Power Daemon.
// It initializes the D-Bus connection, registers managers, and starts the signal server.
func main() {
    // Set log output to standard output.
    log.SetOutput(os.Stdout)

    // Wait for the Power Manager service to initialize.
    log.Println("Waiting for power manager init...")
    time.Sleep(1000 * time.Millisecond)

    // Connect to the system D-Bus.
    log.Println("Trying to connect to the system bus")
    conn, err := dbus.ConnectSystemBus(dbus.WithSignalHandler(dbus.NewSequentialSignalHandler()))
    if err != nil {
        log.Fatalf("Failed to connect to the system bus: %v", err)
    }
    defer conn.Close()

    // Create a context for managing the lifecycle of the daemon.
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Initialize the D-Bus signal server.
    sigServer := dbusutil.NewSignalServer(ctx, conn)

    // Initialize and register the Suspend Manager.
    suspendManager := suspend_manager.NewSuspendManager(ctx, conn)
    if err := suspendManager.Register(sigServer); err != nil {
        log.Fatalf("Failed to register suspend manager: %v", err)
    }
    defer suspendManager.UnRegister(sigServer)

    // Initialize and register the Backlight Manager.
    backlightManager := backlight_manager.NewScreenBrightnessManager(ctx, conn)
    if err := backlightManager.Register(sigServer); err != nil {
        log.Fatalf("Failed to register backlight manager: %v", err)
    }
    defer backlightManager.UnRegister(sigServer)

    // Start the signal server to listen for D-Bus signals.
    sigServer.StartWorking()
}