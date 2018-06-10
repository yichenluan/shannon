package main

import (
	"config"
	"time"
)

type AdaDeal struct {
	AdaInfo   *CoinInfo
	Rebalance *AutoRebalance

	RbChannel chan int
}

func NewAdaDeal() *AdaDeal {
	return &AdaDeal{
		AdaInfo:   NewCoinInfo("ada", config.ShannonConf.AccountID),
		Rebalance: NewARStrategy("ada", config.ShannonConf.AccountID),
		RbChannel: make(chan int, 1),
	}
}

func (ada *AdaDeal) AutoRenew() {
	ada.AdaInfo.RunRenewRoutine()
}

func (ada *AdaDeal) AutoRb() {
	ada.Rebalance.RunRbRountine(ada.RbChannel)
	clocker := time.NewTicker(time.Duration(RENEW_INTERVAL) * time.Millisecond)
	for {
		select {
		case <-clocker.C:
			info := &Info{
				CoinPrice:  ada.AdaInfo.GetCoinPrice(),
				CoinAmount: ada.AdaInfo.GetCoinAmount(),
				USDTAmount: ada.AdaInfo.GetUSDTAmount(),
			}

			ada.Rebalance.ReceiveInfo(info)

		case <-ada.RbChannel:
			// TODO.
			ada.AdaInfo.NeedRenewAmount = true
		}
	}
}
