package pinger4

//Info a
type Info struct {
	IcmpID  int
	Targets map[BinIPv4Address]struct {
		IPAddress string
		Comment   string
	}
	TargetsOrder        []BinIPv4Address
	StatisticsCountsNum int64
	IntervalMillisec    int64
	TimeoutMillisec     int64
	TimeouterCounter    int64
	ResultDropCounter   int64
}

// GetIcmpID is
func (thisPinger *Pinger) GetIcmpID() int {
	return thisPinger.icmpID
}

// GetInfo is
func (thisPinger *Pinger) GetInfo() Info {
	targets := make(map[BinIPv4Address]struct {
		IPAddress string
		Comment   string
	})
	for key, icmpTarget := range thisPinger.targets {
		targets[key] = struct {
			IPAddress string
			Comment   string
		}{
			IPAddress: icmpTarget.ipAddress,
			Comment:   icmpTarget.comment,
		}
	}

	targetsOrder := append(make([]BinIPv4Address, 0, len(thisPinger.targetsOrder)), (thisPinger.targetsOrder)...)

	return Info{
		IcmpID:              thisPinger.icmpID,
		Targets:             targets,
		TargetsOrder:        targetsOrder,
		StatisticsCountsNum: thisPinger.config.StatisticsCountsNum,
		IntervalMillisec:    thisPinger.config.IntervalMillisec,
		TimeoutMillisec:     thisPinger.config.TimeoutMillisec,
		TimeouterCounter:    thisPinger.status.timeouterCounter,
		ResultDropCounter:   thisPinger.status.resultDropCounter,
	}
}
