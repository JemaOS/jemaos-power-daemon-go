package dbusutil

import (
    "context"
    "errors"
    "fmt"

    "github.com/godbus/dbus/v5"
    "github.com/golang/protobuf/proto"
)

// CallProtoMethodWithSequence marshals the input message, sends it as a byte array
// to the specified D-Bus method, and unmarshals the response into the output message.
// It also returns the D-Bus response sequence for tracking purposes.
func CallProtoMethodWithSequence(ctx context.Context, obj dbus.BusObject, method string, in, out proto.Message) (dbus.Sequence, error) {
    var args []interface{}
    if in != nil {
        // Marshal the input protobuf message into a byte array.
        marshIn, err := proto.Marshal(in)
        if err != nil {
            return 0, fmt.Errorf("failed marshaling %s arg", method)
        }
        args = append(args, marshIn)
    }

    // Call the D-Bus method with the marshaled input.
    call := obj.CallWithContext(ctx, method, 0, args...)
    if call.Err != nil {
        return call.ResponseSequence, fmt.Errorf("failed calling %s, err:%w", method, call.Err)
    }

    // If an output message is provided, unmarshal the response into it.
    if out != nil {
        var marshOut []byte
        if err := call.Store(&marshOut); err != nil {
            return call.ResponseSequence, fmt.Errorf("failed reading %s response, err:%w", method, err)
        }
        if err := proto.Unmarshal(marshOut, out); err != nil {
            return call.ResponseSequence, fmt.Errorf("failed unmarshaling %s response, err:%w", method, err)
        }
    }
    return call.ResponseSequence, nil
}

// CallProtoMethod marshals the input protobuf message, sends it to the specified
// D-Bus method, and unmarshals the response into the output message. This is a
// simplified version of CallProtoMethodWithSequence that ignores the response sequence.
func CallProtoMethod(ctx context.Context, obj dbus.BusObject, method string, in, out proto.Message) error {
    _, err := CallProtoMethodWithSequence(ctx, obj, method, in, out)
    return err
}

// DecodeSignal unmarshals the body of a D-Bus signal into the provided protobuf message.
// The signal body must be a byte slice.
func DecodeSignal(sig *dbus.Signal, sigResult proto.Message) error {
    if len(sig.Body) == 0 {
        return errors.New("signal lacked a body")
    }
    buf, ok := sig.Body[0].([]byte)
    if !ok {
        return errors.New("signal body is not a byte slice")
    }
    if err := proto.Unmarshal(buf, sigResult); err != nil {
        return errors.New("failed unmarshaling signal body")
    }
    return nil
}

// GetPMObject returns the D-Bus object for the Power Manager service.
func GetPMObject(conn *dbus.Conn) dbus.BusObject {
    return conn.Object(PowerManagerName, PowerManagerPath)
}

// GetPMMethod constructs the full D-Bus method name by combining the interface
// name with the method name.
func GetPMMethod(method string) string {
    return PowerManagerInterface + "." + method
}