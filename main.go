package main

import (
	"fmt"
	"go-project/processor"
	"go-project/utils"
	"log"
	"sync"
)

func main() {
	filePath := "data/fng.1000.csv.rot128"

	paymentRepo := utils.NewFileReader(filePath)

	paymentServices := processor.NewPaymentService(paymentRepo)

	donations, err := paymentServices.SortDonations()
	if err != nil {
		log.Fatalf("Error sorting donations: %v", err)
	}

	tokenCh := make(chan string)
	chargeCh := make(chan bool)

	var wg sync.WaitGroup

	wg.Add(len(donations))

	for _, d := range donations {
		func(donate processor.Donate) {
			defer wg.Done()

			tokenID, _, err := paymentServices.CreateTokenAndCharge([]processor.Donate{donate})
			if err != nil {
				fmt.Printf("Error creating token: %v\n", err)
				tokenCh <- ""
				return
			}

			if len(tokenID) == 0 {
				fmt.Println("Token ID not available")
				tokenCh <- ""
				return
			}

			tokenCh <- tokenID[0]

			// Create charge
			chargePaid, err := paymentServices.CreateAndCheckCharge(tokenID[0], donate.Amount)
			if err != nil {
				fmt.Printf("Error creating or checking charge: %v\n", err)
				chargeCh <- false
				return
			}
			chargeCh <- chargePaid
		}(d)
	}

	go func() {
		for tokenID := range tokenCh {
			if tokenID == "" {
				fmt.Println("Token creation failed")
				continue
			}
			fmt.Printf("Token ID: %s\n", tokenID)
		}
	}()

	go func() {
		for chargePaid := range chargeCh {
			fmt.Printf("Charge Paid: %t\n", chargePaid)
		}
	}()

	wg.Wait()

	close(tokenCh)
	close(chargeCh)
}
