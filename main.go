package main

import (
	"config"
	"korok"
	"sync"
)

func main() {
	err := config.GetShannonConfig("../shannon_conf/shannon.conf")
	if err != nil {
		korok.Fatal("GetShannonConfig Failed: %s", err)
		return
	}

	var wait sync.WaitGroup
	ar := NewAR()
	ar.Run()
	korok.Info("AutoRebalance Begin to Run.")

	wait.Add(1)
	wait.Wait()
}
