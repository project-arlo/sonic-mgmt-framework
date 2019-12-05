package transformer

import (
	// Go 1.11 doesn't let us download using the canonical import path
	// "github.com/godbus/dbus/v5"
	// This works around that problem by using gopkg.in instead
	"gopkg.in/godbus/dbus.v5"
)

// hostAccountCallObject returns a dbus.BusObject which can be used to call
// the requested method
func hostAccountCallObject(method string) (dbus.BusObject, string, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, "", err
	}

	const bus_name_base = "org.SONiC.HostAccountManagement."
	bus_name := bus_name_base + method
	bus_path := dbus.ObjectPath("/org/SONiC/HostAccountManagement/" + method)

	obj := conn.Object(bus_name, bus_path)
	dest := bus_name_base + method

	return obj, dest, nil
}

func hostAccountParseCallReturn(call *dbus.Call) (bool, string) {
	if call.Err != nil {
		return false, call.Err.Error()
	}

	success := call.Body[0].(bool)
	errmsg := call.Body[1].(string)

	return success, errmsg
}

// hostAccountUserAdd calls the HAM useradd function over D-Bus
func hostAccountUserAdd(login, role, hashed_pw string) (bool, string) {
	obj, dest, err := hostAccountCallObject("useradd")
	if err != nil {
		return false, err.Error()
	}

	return hostAccountParseCallReturn(obj.Call(dest, 0, login, role, hashed_pw))
}

// hostAccountUserDel calls the HAM userdel over D-Bus
func hostAccountUserDel(login string) (bool, string) {
	obj, dest, err := hostAccountCallObject("userdel")
	if err != nil {
		return false, err.Error()
	}

	return hostAccountParseCallReturn(obj.Call(dest, 0, login))
}

// hostAccountChPasswd calls the HAM chpasswd over D-Bus
func hostAccountChPasswd(login, hashed_pw string) (bool, string) {
	obj, dest, err := hostAccountCallObject("chpasswd")
	if err != nil {
		return false, err.Error()
	}

	return hostAccountParseCallReturn(obj.Call(dest, 0, login, hashed_pw))
}

// hostAccountChRole calls the HAM chrole over D-Bus
func hostAccountChRole(login, role string) (bool, string) {
	obj, dest, err := hostAccountCallObject("chrole")
	if err != nil {
		return false, err.Error()
	}

	return hostAccountParseCallReturn(obj.Call(dest, 0, login, role))
}
