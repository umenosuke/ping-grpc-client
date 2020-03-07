package main

import (
	"encoding/json"
	"io/ioutil"
)

// Config 設定ファイルの中身
type Config struct {
}

// DefaultConfig is return default value config
func DefaultConfig() Config {
	return Config{}
}

func configLoad(path string) (Config, error) {
	res := DefaultConfig()

	if path == "" {
		return res, nil
	}

	jsonString, err := ioutil.ReadFile(path)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(jsonString, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func configStringify(data Config) string {
	jsonBlob, _ := json.MarshalIndent(data, "", "")

	return string(jsonBlob)
}
