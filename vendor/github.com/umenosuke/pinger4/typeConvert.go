package pinger4

import (
	"encoding/binary"
	"net"
	"strconv"
)

//NetIP2BinIPv4Address a
func NetIP2BinIPv4Address(ip net.IP) BinIPv4Address {
	if len(ip) == 16 {
		return BinIPv4Address(binary.BigEndian.Uint32(ip[12:16]))
	}
	return BinIPv4Address(binary.BigEndian.Uint32(ip))
}

//BinIPv4Address2String a
func BinIPv4Address2String(ipBin BinIPv4Address) string {
	return "" + strconv.FormatUint(uint64((ipBin>>24)&0xff), 10) + "." + strconv.FormatUint(uint64((ipBin>>16)&0xff), 10) + "." + strconv.FormatUint(uint64((ipBin>>8)&0xff), 10) + "." + strconv.FormatUint(uint64((ipBin>>0)&0xff), 10)
}
