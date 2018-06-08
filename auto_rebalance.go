package main

import (
	"config"
	"korok"
	"services"
)

func main() {
	err := config.GetShannonConfig("../shannon_conf/shannon.conf")
	if err != nil {
		korok.Fatal("GetShannonConfig Failed: %s", err)
		return
	}
	balance, err := services.GetAccountBalance(config.ShannonConf.AccountID)

	if err == nil {
		korok.Info("balance: %v", balance)
	}
}
