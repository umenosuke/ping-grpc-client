package main

import (
	"encoding/json"
	"io/ioutil"
)

// Config 設定ファイルの中身
type Config struct {
	//pingを撃ち続ける時間(秒)
	StopPingerSec uint64 `json:"StopPingerSec"`

	//一つの対象へのpingを撃つインターバル(ミリ秒)
	IntervalMillisec uint64 `json:"IntervalMillisec"`

	//pingのタイムアウトまでの時間(ミリ秒)
	TimeoutMillisec uint64 `json:"TimeoutMillisec"`

	//pingの統計をとるため、過去いくつの結果を保持するか
	StatisticsCountsNum uint64 `json:"StatisticsCountsNum"`

	//pingの統計を集計するインターバル
	StatisticsIntervalSec uint64 `json:"StatisticsIntervalSec"`

	//pingの統計表示で正常レスポンスが何％以上を成功とするか
	CountRateThreshold int64 `json:"CountRateThreshold"`
}

// DefaultConfig is return default value config
func DefaultConfig() Config {
	return Config{
		StopPingerSec:         3600 * 4,
		IntervalMillisec:      1000,
		TimeoutMillisec:       1000,
		StatisticsCountsNum:   10,
		StatisticsIntervalSec: 1,
		CountRateThreshold:    80,
	}
}

func configLoad(configPath string, configJSON string) (Config, error) {
	res := DefaultConfig()

	if configPath != "" {
		jsonString, err := ioutil.ReadFile(configPath)
		if err != nil {
			return res, err
		}
		err = json.Unmarshal(jsonString, &res)
		if err != nil {
			return res, err
		}
	}

	err := json.Unmarshal([]byte(configJSON), &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func configStringify(data Config) string {
	jsonBlob, _ := json.MarshalIndent(data, "", "")

	return string(jsonBlob)
}
