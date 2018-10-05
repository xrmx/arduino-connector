// Code generated by "stringer -type=NmState"; DO NOT EDIT

package gonetworkmanager

import "fmt"

const (
	_NmState_name_0 = "NmStateUnknown"
	_NmState_name_1 = "NmStateAsleep"
	_NmState_name_2 = "NmStateDisconnected"
	_NmState_name_3 = "NmStateDisconnecting"
	_NmState_name_4 = "NmStateConnecting"
	_NmState_name_5 = "NmStateConnectedLocal"
	_NmState_name_6 = "NmStateConnectedSite"
	_NmState_name_7 = "NmStateConnectedGlobal"
)

var (
	_NmState_index_0 = [...]uint8{0, 14}
	_NmState_index_1 = [...]uint8{0, 13}
	_NmState_index_2 = [...]uint8{0, 19}
	_NmState_index_3 = [...]uint8{0, 20}
	_NmState_index_4 = [...]uint8{0, 17}
	_NmState_index_5 = [...]uint8{0, 21}
	_NmState_index_6 = [...]uint8{0, 20}
	_NmState_index_7 = [...]uint8{0, 22}
)

func (i NmState) String() string {
	switch {
	case i == 0:
		return _NmState_name_0
	case i == 10:
		return _NmState_name_1
	case i == 20:
		return _NmState_name_2
	case i == 30:
		return _NmState_name_3
	case i == 40:
		return _NmState_name_4
	case i == 50:
		return _NmState_name_5
	case i == 60:
		return _NmState_name_6
	case i == 70:
		return _NmState_name_7
	default:
		return fmt.Sprintf("NmState(%d)", i)
	}
}