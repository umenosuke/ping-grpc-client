package pinger4

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/umenosuke/labelinglog"
)

// Pinger is Pinger
type Pinger struct {
	config Config
	logger *labelinglog.LabelingLogger

	icmpID       int
	targets      map[BinIPv4Address]icmpTarget
	targetsOrder []BinIPv4Address

	reqList map[BinIPv4Address]*struct {
		sync.Mutex
		req [responseListNum]struct {
			isReceived bool
		}
	}

	isStarted struct {
		sync.Mutex
		flg bool
	}
	cancelFunc context.CancelFunc

	chIcmpResponse        chan icmpResponse
	chIcmpResponseTimeout chan icmpResponseTimeout
	chIcmpResult          chan IcmpResult

	statisticsData struct {
		targets map[BinIPv4Address]*sData
	}

	timeouterList map[BinIPv4Address]*struct {
		sync.Mutex
		cancelFunc [responseListNum]context.CancelFunc
	}

	status struct {
		timeouterCounter  int64
		resultDropCounter int64
	}

	chIcmpResultsSubscriber []chan IcmpResult

	addIntervalVar struct {
		sync.Mutex
		lastExecutionTime int64
	}
}

// New is create Pinger
func New(icmpID int, pingerConfig Config) Pinger {
	return Pinger{
		config: pingerConfig,
		logger: labelinglog.New("pinger "+strconv.Itoa(icmpID), os.Stderr),

		icmpID:       icmpID,
		targets:      make(map[BinIPv4Address]icmpTarget),
		targetsOrder: make([]BinIPv4Address, 0),
		reqList: make(map[BinIPv4Address]*struct {
			sync.Mutex
			req [responseListNum]struct {
				isReceived bool
			}
		}),

		isStarted: struct {
			sync.Mutex
			flg bool
		}{
			flg: false,
		},
		cancelFunc: func() {},

		chIcmpResponse:        make(chan icmpResponse, pingerConfig.MaxWorkers*2),
		chIcmpResponseTimeout: make(chan icmpResponseTimeout, pingerConfig.MaxWorkers*2),
		chIcmpResult:          make(chan IcmpResult, pingerConfig.MaxWorkers*2),

		statisticsData: struct {
			targets map[BinIPv4Address]*sData
		}{
			targets: make(map[BinIPv4Address]*sData),
		},

		timeouterList: make(map[BinIPv4Address]*struct {
			sync.Mutex
			cancelFunc [responseListNum]context.CancelFunc
		}),

		status: struct {
			timeouterCounter  int64
			resultDropCounter int64
		}{
			timeouterCounter:  0,
			resultDropCounter: 0,
		},

		chIcmpResultsSubscriber: make([]chan IcmpResult, 0),

		addIntervalVar: struct {
			sync.Mutex
			lastExecutionTime int64
		}{
			lastExecutionTime: time.Now().UnixNano(),
		},
	}
}

func (thisPinger *Pinger) addInterval() {
	now := time.Now().UnixNano()
	thisPinger.addIntervalVar.Lock()
	defer thisPinger.addIntervalVar.Unlock()
	if thisPinger.addIntervalVar.lastExecutionTime+(5*1000*1000*1000) < now {
		thisPinger.addIntervalVar.lastExecutionTime = now
		oldInterval := thisPinger.config.IntervalMillisec
		thisPinger.config.IntervalMillisec = oldInterval * 2
		thisPinger.logger.Log(labelinglog.FlgWarn, "pinger busy, interval change ["+strconv.FormatInt(oldInterval, 10)+"ms to "+strconv.FormatInt(thisPinger.config.IntervalMillisec, 10)+"ms]")
	}
}

//BinIPv4Address a
type BinIPv4Address uint32
