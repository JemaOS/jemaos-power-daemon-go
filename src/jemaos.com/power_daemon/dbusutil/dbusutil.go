package dbusutil

import (
  "context"
  "errors"
  "fmt"
  "github.com/godbus/dbus/v5"
  "github.com/golang/protobuf/proto"
)

func CallProtoMethodWithSequence(ctx context.Context, obj dbus.BusObject, method string, in, out proto.Message) (dbus.Sequence, error) {
  var args []interface{}
  if in != nil {
    marshIn, err := proto.Marshal(in)
    if err != nil {
      return 0, fmt.Errorf("failed marshaling %s arg", method)
    }
    args = append(args, marshIn)
  }

  call := obj.CallWithContext(ctx, method, 0, args...)
  if call.Err != nil {
    return call.ResponseSequence, fmt.Errorf("failed calling %s, err:%w", method, call.Err)
  }
  if out != nil {
    var marshOut []byte
    if err := call.Store(&marshOut); err != nil {
      return call.ResponseSequence, fmt.Errorf( "failed reading %s response, err:%w", method, err)
    }
    if err := proto.Unmarshal(marshOut, out); err != nil {
      return call.ResponseSequence, fmt.Errorf("failed unmarshaling %s response, err:%w", method, err)
    }
  }
  return call.ResponseSequence, nil
}

// CallProtoMethod marshals in, passes it as a byte array arg to method on obj,
// and unmarshals a byte array arg from the response to out. method should be prefixed
// by a D-Bus interface name. Both in and out may be nil.
func CallProtoMethod(ctx context.Context, obj dbus.BusObject, method string, in, out proto.Message) error {
  _, err := CallProtoMethodWithSequence(ctx, obj, method, in, out)
  return err
}

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

func GetPMObject(conn *dbus.Conn) dbus.BusObject {
  return conn.Object(PowerManagerName, PowerManagerPath)
}

func GetPMMethod(method string) string {
  return PowerManagerInterface + "." + method
}
