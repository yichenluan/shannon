package main

import (
	"config"
	"korok"
	"services"
	"strconv"
	"sync"
	"errors"
	"encoding/json"
	"time"
)

const (
	RENEW_INTERVAL = 1000 //ms
)

func NewCoinInfo(name string, accountID string) *CoinInfo {
	return &CoinInfo{
		CoinName : name,
		AccountID: accountID,
		NeedRenewAmount : true,
	}
}

type CoinInfo struct {
	CoinName 	string
	AccountID 	string

	NeedRenewAmount 	bool

	Mu	sync.Mutex

	CoinPrice 	float64
	CoinAmount  float64
	USDTAmount float64
}

func (ci *CoinInfo) RunRenewRoutine() {
	go ci.ClockRenew()
}

func (ci *CoinInfo) ClockRenew() {
	clocker := time.NewTicker(time.Duration(RENEW_INTERVAL) * time.Millisecond)
	for {
		select {
		case <- clocker.C:
			if ci.NeedRenewAmount {
				err := ci.RenewAmountInfo()
				if err == nil {
					ci.NeedRenewAmount = false
				}
			}
			ci.RenewPriceInfo()
		}
	}
}

func (ci *CoinInfo) SetCoinAmount(amount float64) {
	ci.Mu.Lock()
	ci.CoinAmount = amount
	ci.Mu.Unlock()
}

func (ci *CoinInfo) SetUSDTAmount(amount float64) {
	ci.Mu.Lock()
	ci.USDTAmount = amount
	ci.Mu.Unlock()
}

func (ci *CoinInfo) GetCoinAmount() float64 {
	ci.Mu.Lock()
	defer ci.Mu.Unlock()
	return ci.CoinAmount
}

func (ci *CoinInfo) GetUSDTAmount() float64 {
	ci.Mu.Lock()
	defer ci.Mu.Unlock()
	return ci.USDTAmount
}

func (ci *CoinInfo) SetCoinPrice(price float64) {
	ci.Mu.Lock()
	ci.CoinPrice = price
	ci.Mu.Unlock()
}

func (ci *CoinInfo) GetCoinPrice() float64 {
	ci.Mu.Lock()
	defer ci.Mu.Unlock()

	return ci.CoinPrice
}

func (ci *CoinInfo) RenewAmountInfo() error {
	balance, err := services.GetAccountBalance(ci.AccountID)
	if err != nil {
		korok.Fatal("GetAccountBalance Failed : %s", err)
		return err
	}

	balanceList := balance.Data.List
	for _, sub := range balanceList {
		if sub.Type != "trade" {
			continue
		}
		if sub.Currency == "usdt" {
			//ar.USDTAmount = float64(sub.Balance)
			if f, err := strconv.ParseFloat(sub.Balance, 64); err == nil {
				ci.SetUSDTAmount(f)
			} else {
				korok.Fatal("ParseFloat to USDTAmount Failed, string: %s", sub.Balance)
				return err
			}
		} else if sub.Currency == config.ShannonConf.Coin {
			if f, err := strconv.ParseFloat(sub.Balance, 64); err == nil {
				ci.SetCoinAmount(f)
			} else {
				korok.Fatal("ParseFloat to CoinAmount Failed, string: %s", sub.Balance)
				return err
			}
		}
	}
	return nil

}

func (ci *CoinInfo) RenewPriceInfo() error {
	symbol := ci.CoinName + "usdt"
	price, err := services.GetKLine(symbol, "1min", 1)
	if err != nil {
		korok.Fatal("GetKLine Failed : %s", err)
		return err
	}

	kLineData := price.Data
	if len(kLineData) != 1 {
		korok.Fatal("kLineData len != 1")
		return errors.New("kLineData len != 1")
	}

	kLine := kLineData[0]

	//kLineStr, _ := json.Marshal(kLine)
	//korok.Info("kLine : %v", string(kLineStr))
	ci.SetCoinPrice(kLine.Close)
	korok.Info("Curr %s Price: %v", ci.CoinName, kLine.Close)

	return nil
}


type Strategy interface {}

type AdaDeal struct {
	AdaInfo 	*CoinInfo
	strategy 	Strategy
}

func NewAdaDeal() *AdaDeal {
	return &AdaDeal{
		AdaInfo: NewCoinInfo("ada", config.ShannonConf.AccountID),
	}
}

func (ada *AdaDeal) AutoRenew() {
	ada.AdaInfo.RunRenewRoutine()
}

func (ada *AdaDeal) Clock() {
	clocker := time.NewTicker(time.Duration(5000) * time.Millisecond)
	for {
		select {
		case <- clocker.C:
			ada.AdaInfo.NeedRenewAmount = true
		}
	}
}