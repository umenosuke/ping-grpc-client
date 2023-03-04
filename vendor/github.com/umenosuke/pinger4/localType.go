package pinger4

import (
	"net"
	"sync"

	"golang.org/x/net/icmp"
)

type icmpTarget struct {
	id           BinIPv4Address
	ipAddress    string
	comment      string
	binIPAddress net.IP
	netIPAddr    *net.IPAddr
}

type icmpRequestQueueEntry struct {
	icmpTargetID BinIPv4Address
	seq          int
}

type icmpRequest struct {
	icmpTargetID    BinIPv4Address
	seq             int
	sendTimeNanosec int64
}

type icmpResponse struct {
	peer               net.Addr
	message            *icmp.Message
	receiveTimeNanosec int64
}

type icmpResponseTimeout struct {
	icmpTargetID       BinIPv4Address
	seq                int
	sendTimeNanosec    int64
	receiveTimeNanosec int64
}

type sData struct {
	sync.Mutex
	Res   []int64
	Index int64
}

type tICMPData struct {
	IcmpTargetID    uint32
	SendTimeNanosec int64
}
