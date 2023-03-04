package pinger4

const terminateTimeOutSec = 10
const responseListNum = 0x1000
const responseMTU = 1500
const limitTimeouter = 100000

// Config a
type Config struct {
	DebugEnable            bool   `json:"DebugEnable"`
	DebugPrintIntervalSec  int64  `json:"DebugPrintInterval"`
	SourceIPAddress        string `json:"SourceIPAddress"`
	StartSendIcmpSmoothing bool   `json:"StartSendIcmpSmoothing"`
	IntervalMillisec       int64  `json:"IntervalMillisec"`
	TimeoutMillisec        int64  `json:"TimeoutMillisec"`
	MaxWorkers             int64  `json:"MaxWorkers"`
	StatisticsCountsNum    int64  `json:"StatisticsCountsNum"`
}

// DefaultConfig a
func DefaultConfig() Config {
	return Config{
		DebugEnable:            false,
		DebugPrintIntervalSec:  1,
		SourceIPAddress:        "0.0.0.0",
		StartSendIcmpSmoothing: true,
		IntervalMillisec:       500,
		TimeoutMillisec:        1000,
		MaxWorkers:             1000,
		StatisticsCountsNum:    50,
	}
}
