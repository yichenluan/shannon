package main

import (
	"config"
	"korok"
)

func main() {
	err := config.GetShannonConfig("../shannon_conf/shannon.conf")
	if err != nil {
		korok.Fatal("GetShannonConfig Failed: %s", err)
		return
	}

	ada := NewAdaDeal()

	ada.AutoRenew()
	ada.AutoRb()
}
