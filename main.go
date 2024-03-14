package main

import (
	"fmt"
	"go-project/processor"
	"go-project/utils"
	"log"
)

func main() {
	filePath := `data\fng.1000.csv.rot128`

	paymentRepo := utils.NewFileReader(filePath)
	paymentServices := processor.NewPaymentService(paymentRepo)

	donations, err := paymentServices.SortDonations()
	if err != nil {
		log.Fatalf("Error sorting donations: %v", err)
	}

	chargeIDs, err := paymentServices.CreatePaymentToken(donations)
	if err != nil {
		log.Fatalf("Error creating payment tokens: %v", err)
	}

	donateInfo, err := paymentServices.CalculateDonate(donations)
	if err != nil {
		log.Fatalf("Error creating payment tokens: %v", err)
	}

	fmt.Printf("Performing donations...\n")
	fmt.Printf("done.\n\n")
	fmt.Printf("\t       Total Donations Processed: %d\n", len(chargeIDs))
	fmt.Printf("\tSuccessfully donated: THB  %d\n", donateInfo.ValidSum)
	fmt.Printf("\t          Top donest: THB     %d\n", donateInfo.TopDonate)
}
