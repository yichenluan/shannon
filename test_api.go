package main

import "fmt"
import "services"

func main() {
	//accounts := services.GetAccounts()
	//fmt.Printf("accounts: %v\n", accounts)

	data := services.GetAccountBalance("")
	fmt.Printf("data: %v\n", data)
}
