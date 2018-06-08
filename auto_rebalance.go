package main

import (
	"config"
	"korok"
	"services"
	"strconv"
	"sync"
	"time"
)

func NewAR() *AutoRebalance {
	return &AutoRebalance{}
}

type AutoRebalance struct {
	CoinPrice float64

	AmountMu   sync.Mutex
	CoinAmount float64
	USDTAmount float64
}

func (ar *AutoRebalance) Run() {
	go ar.FetchCurrInfo()
}

func (ar *AutoRebalance) FetchCurrInfo() {
	clocker := time.NewTicker(time.Duration(1000) * time.Millisecond)

	for {
		select {
		case <-clocker.C:
			err := ar.FetchAmountInfo()
			if err == nil {
				korok.Info("FetchAmountInfo Success. CoinAmount: %f, USDTAmount: %f", ar.GetCoinAmount(), ar.GetUSDTAmount())
			}
		}
	}
}

func (ar *AutoRebalance) GetCoinAmount() float64 {
	ar.AmountMu.Lock()
	defer ar.AmountMu.Unlock()

	return ar.CoinAmount
}

func (ar *AutoRebalance) GetUSDTAmount() float64 {
	ar.AmountMu.Lock()
	defer ar.AmountMu.Unlock()

	return ar.USDTAmount
}

func (ar *AutoRebalance) FetchAmountInfo() error {
	balance, err := services.GetAccountBalance(config.ShannonConf.AccountID)
	if err != nil {
		korok.Fatal("Fetch Amount Info Failed.")
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
				ar.USDTAmount = f
			} else {
				korok.Fatal("ParseFloat to USDTAmount Failed, string: %s", sub.Balance)
				return err
			}
		} else if sub.Currency == config.ShannonConf.Coin {
			if f, err := strconv.ParseFloat(sub.Balance, 64); err == nil {
				ar.CoinAmount = f
			} else {
				korok.Fatal("ParseFloat to CoinAmount Failed, string: %s", sub.Balance)
				return err
			}
		}
	}
	return nil
}
