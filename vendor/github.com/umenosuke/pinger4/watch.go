package pinger4

//IcmpResultType a
type IcmpResultType uint8

//IcmpResult a
type IcmpResult struct {
	ResultType             IcmpResultType
	IcmpTargetID           BinIPv4Address
	BinPeerIP              BinIPv4Address
	Seq                    int
	SendTimeUnixNanosec    int64
	ReceiveTimeUnixNanosec int64
}

//IcmpResultType a
const (
	IcmpResultUnknown = IcmpResultType(iota)
	IcmpResultTypeReceive
	IcmpResultTypeReceiveAfterTimeout
	IcmpResultTypeTTLExceeded
	IcmpResultTypeTimeout
)

//GetChIcmpResult a
func (thisPinger *Pinger) GetChIcmpResult(cap int) <-chan IcmpResult {
	ch := make(chan IcmpResult, cap)
	thisPinger.chIcmpResultsSubscriber = append(thisPinger.chIcmpResultsSubscriber, ch)

	return ch
}

//SuccessCounts a
type SuccessCounts map[BinIPv4Address]struct {
	Count int64
}

// GetSuccessCounts a
func (thisPinger *Pinger) GetSuccessCounts() SuccessCounts {
	count := make(SuccessCounts)

	for id, target := range thisPinger.statisticsData.targets {
		target.Lock()
		defer target.Unlock()

		sum := int64(0)
		for _, list := range target.Res {
			sum += list
		}
		count[id] = struct {
			Count int64
		}{
			Count: sum,
		}
	}

	return count
}
