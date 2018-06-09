package main

import (
	"time"
	"services"
	"fmt"
	"errors"
	"models"
	"config"
	"korok"
)

type Info struct {
	CoinPrice 	float64
	CoinAmount 	float64
	USDTAmount	float64
}

const (
	ACTION_NONEED = iota
	ACTION_SELL
	ACTION_BUY
)

func NewARStrategy(name string, accountID string) *AutoRebalance {
	return &AutoRebalance {
		CoinName: name,
		AccountID: accountID,
		PerfectRatio: config.ShannonConf.PerfectRatio,
		UpRatio : config.ShannonConf.UpRatio,
		DownRatio : config.ShannonConf.DownRatio,
		InfoChannel : make(chan *Info, 100),
	}
}

type AutoRebalance struct {
	CoinName 			string
	AccountID 			string

	LastRbTime 			time.Time
	LastRbCoinPrice		float64
	LastRbCoinAmount 	float64
	LastRbUSDTAmount 	float64

	PerfectRatio 		float64

	UpRatio				float64
	DownRatio 			float64

	InfoChannel 		chan *Info
}

func (ar *AutoRebalance) ReceiveInfo(info *Info) {
	ar.InfoChannel <- info
}

func (ar *AutoRebalance) RunRbRountine(Signal chan int) {
	go ar.AutoRb(Signal)
}

func (ar *AutoRebalance) AutoRb(Signal chan int) {
	for {
		select {
		case info := <- ar.InfoChannel:
			opRecord, isChange := ar.HandleInfo(info)
			if isChange {
				mailHead := fmt.Sprintf("[BlockChain] %s Rebalance Happend !!", ar.CoinName)
				go SendMail(mailHead, opRecord)
				Signal <- 1
			}
		}
	}
}

func (ar * AutoRebalance) HandleInfo(info *Info) (opRecord string, isChange bool) {
	ratio, err := ar.CurrRatio(info)
	if err != nil {
		opRecord = "Compute CurrRatio Failed."
		return opRecord, true
	}
	action := ar.RbAction(ratio)
	if action == ACTION_NONEED {
		isChange = false
		return
	}
	totalAsset := info.CoinPrice * info.CoinAmount + info.USDTAmount
	perfectCoinAsset := totalAsset * (ar.PerfectRatio / (ar.PerfectRatio + 1))

	var placeErr error
	if action == ACTION_SELL {
		coinSellAsset := info.CoinAmount * info.CoinPrice - perfectCoinAsset
		coinSellAmount := coinSellAsset / info.CoinPrice

		opRecord += fmt.Sprintf("<h1>SELL %s HAPPEND !</h1>\n\n", ar.CoinName)
		opRecord += fmt.Sprintf("<h2>SELL INFO</h2>\n")
		opRecord += fmt.Sprintf("SELL COIN: %s\n", ar.CoinName)
		opRecord += fmt.Sprintf("SELL AMOUNT: %f\n", coinSellAmount)
		opRecord += fmt.Sprintf("SELL PRICE: %f\n", info.CoinPrice)
		opRecord += fmt.Sprintf("SELL ASSET: %f\n\n", coinSellAsset)

		placeErr = ar.SellCoin(coinSellAmount)
	} else if action == ACTION_BUY {
		coinBuyAsset := perfectCoinAsset - info.CoinAmount * info.CoinPrice
		coinBuyAmount := coinBuyAsset / info.CoinPrice

		opRecord += fmt.Sprintf("<h1>BUY %s HAPPEND !</h1>\n\n", ar.CoinName)
		opRecord += fmt.Sprintf("<h2>BUY INFO</h2>\n")
		opRecord += fmt.Sprintf("BUY COIN: %s\n", ar.CoinName)
		opRecord += fmt.Sprintf("BUY AMOUNT: %f\n", coinBuyAmount)
		opRecord += fmt.Sprintf("BUY PRICE: %f\n", info.CoinPrice)
		opRecord += fmt.Sprintf("BUY ASSET: %f\n\n", coinBuyAsset)

		placeErr = ar.BuyCoin(coinBuyAmount)
	}

	if placeErr != nil {
		isChange = false
		return
	}

	ar.LastRbTime = time.Now()
	ar.LastRbCoinPrice = info.CoinPrice
	ar.LastRbCoinAmount = info.CoinAmount
	ar.LastRbUSDTAmount = info.USDTAmount

	opRecord += fmt.Sprintf("<h2>BEFORE SELL/BUY INFO</h2>\n")
	opRecord += fmt.Sprintf("BEFORE COIN AMOUNT: %f\n", info.CoinAmount)
	opRecord += fmt.Sprintf("BEFORE COIN ASSET: %f\n", info.CoinAmount * info.CoinPrice)
	opRecord += fmt.Sprintf("BEFORE USDT ASSET: %f\n", info.USDTAmount)
	opRecord += fmt.Sprintf("BEFORE %s/usdt RATIO: %f", ar.CoinName, info.CoinAmount * info.CoinPrice / info.USDTAmount)

	return opRecord, true
}

func (ar *AutoRebalance) BuyCoin(amount float64) error {
	return nil

	buyPara := models.PlaceRequestParams {
		AccountID: ar.AccountID,
		Amount:   fmt.Sprintf("%v", amount),
		Source: "api",
		Symbol: ar.CoinName + "usdt",
		Type: "buy-market",
	}
	res, err := services.Place(buyPara)
	if err != nil {
		korok.Fatal("Place Buy Faild: %s", err)
		return err
	}

	if res.Status != "ok" {
		korok.Fatal("Place Buy Faild with ErrCode: %s, ErrMsg: %s", res.ErrCode, res.ErrMsg)
		return errors.New(fmt.Sprintf("Place Buy Faild with ErrCode: %s, ErrMsg: %s", res.ErrCode, res.ErrMsg))
	}

	return nil
}

func (ar *AutoRebalance) SellCoin(amount float64) error {
	return nil
	
	sellPara := models.PlaceRequestParams {
		AccountID: ar.AccountID,
		Amount: fmt.Sprintf("%v", amount),
		Source: "api",
		Symbol: ar.CoinName + "usdt",
		Type: "sell-market",
	}
	res, err := services.Place(sellPara)
	if err != nil {
		korok.Fatal("Place Sell Faild: %s", err)
		return err
	}

	if res.Status != "ok" {
		korok.Fatal("Place Sell Faild with ErrCode: %s, ErrMsg: %s", res.ErrCode, res.ErrMsg)
		return errors.New(fmt.Sprintf("Place Sell Faild with ErrCode: %s, ErrMsg: %s", res.ErrCode, res.ErrMsg))
	}
	
	return nil
}

func (ar *AutoRebalance) CurrRatio(info *Info) (float64, error) {
	if (info.CoinAmount <= 0 || info.USDTAmount <= 0 || info.CoinPrice <= 0) {
		korok.Fatal("Amount Error, CoinPrice: %f, CoinAmount: %f, USDTAmount: %f", info.CoinPrice, info.CoinAmount, info.USDTAmount)
		return 0, errors.New("Amount Error")
	}
	return info.CoinPrice * info.CoinAmount / info.USDTAmount, nil
}

func (ar *AutoRebalance) RbAction(ratio float64) int {
	if ratio > ar.UpRatio {
		return ACTION_SELL
	} else if ratio < ar.DownRatio {
		return ACTION_BUY
	}
	return ACTION_NONEED
}