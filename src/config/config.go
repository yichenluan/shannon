package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

var (
	ACCESS_KEY  string
	SECRET_KEY  string
	ShannonConf *ShannonConfig
)

// API请求地址, 不要带最后的/
const (
	MARKET_URL string = "https://api.huobi.pro"
	TRADE_URL  string = "https://api.huobi.pro"
)

type ShannonConfig struct {
	AccessKey string  `json:"AccessKey"`
	SecretKey string  `json:"SecretKey"`
	AccountID string  `json:"AccountID"`
	Ratio     float32 `json:"Ratio"`
	Coin      string  `json:"Coin"`
}

func GetShannonConfig(path string) error {
	res := &ShannonConfig{}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()

	context, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(context, res)
	if err != nil {
		return err
	}

	ShannonConf = res
	ACCESS_KEY = res.AccessKey
	SECRET_KEY = res.SecretKey
	return nil
}
