package main

import (
	"korok"
	"services"
	"strconv"
	"sync"
	"errors"
	//"encoding/json"
	"time"
	"fmt"
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
	go ci.ClockMail()
}

func (ci *CoinInfo) ClockMail() {
	clocker := time.NewTicker(time.Duration(1) * time.Minute)
	for {
		select {
		case <- clocker.C:
			mailHead := "[BlockChain] Ticker Inform"
			mailBody := ci.CoinInfoBody(mailHead)
			go SendMail(mailHead, mailBody)
		}
	}
}

func (ci *CoinInfo) ClockRenew() {
	clocker := time.NewTicker(time.Duration(RENEW_INTERVAL) * time.Millisecond)
	for {
		select {
		case <- clocker.C:
			ci.RenewAmountInfo()
			err := ci.RenewPriceInfo()
			if err == nil {
				korok.Info("[Price Info] %s price: %f.", ci.CoinName, ci.GetCoinPrice())
			}
			if ci.NeedRenewAmount {
				korok.Info("[Amount Info] %s amount: %f, usdt amount: %f.", ci.CoinName, ci.GetCoinAmount(), ci.GetUSDTAmount())
				ci.NeedRenewAmount = false
				mailHead := "[BlockChain] Renew Inform !!!"
				mailBody := ci.CoinInfoBody(mailHead)
				go SendMail(mailHead, mailBody)
			}
		}
	}
}

func (ci *CoinInfo) CoinInfoBody(head string) (body string) {
	coinAmount := ci.GetCoinAmount()
	coinPrice := ci.GetCoinPrice()
	usdtAmount := ci.GetUSDTAmount()

	body += head
	body += fmt.Sprintf("\n\nCOIN: %s\n", ci.CoinName)
	body += fmt.Sprintf("COIN AMOUNT: %f, COIN PRICE: %f\n", coinAmount, coinPrice)
	body += fmt.Sprintf("COIN ASSET: %f, USDT ASSET: %f\n", coinAmount * coinPrice, usdtAmount)
	body += fmt.Sprintf("COIN / USDT RATIO: %s/usdt: %f\n", ci.CoinName, coinAmount*coinPrice / usdtAmount)
	return body
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
		} else if sub.Currency == ci.CoinName {
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

	ci.SetCoinPrice(kLine.Close)

	return nil
}