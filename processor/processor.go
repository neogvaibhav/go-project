package processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-project/utils"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DonateInfo struct {
	InvalidSum  int
	ValidSum    int
	TotalSum    int
	InvalidCard int
	ValidCard   int
	TotalCard   int
	TopDonate   int
	TopDonor    string
}

type Donate struct {
	Amount   int
	CcNumber string
	CVV      string
	Card     Card
}

type Card struct {
	Name            string
	ExpirationMonth int
	ExpirationYear  int
}

type CardDetails struct {
	Name            string `json:"Name"`
	City            string `json:"city"`
	PostalCode      string `json:"postal_code"`
	Number          string `json:"number"`
	SecurityCode    string `json:"security_code"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
}

type TokenResponse struct {
	ID string `json:"id"`
}

type ChargeInfo struct {
	Description string `json:"description"`
	Amount      int    `json:"Amount"`
	Currency    string `json:"currency"`
	ReturnURI   string `json:"return_uri"`
	Card        string `json:"Card"`
}

type ChargeResponse struct {
	ID string `json:"id"`
}

const secretKey = "skey_test_5z2kz7ytse0t03341hs"

type PaymentService interface {
	SortDonations() (donates []Donate, err error)
	CalculateDonate(donates []Donate) (DonateInfo DonateInfo, err error)
	CreatePaymentToken(donates []Donate) ([]string, error)
	CreateTokenAndCharge(donates []Donate) ([]string, []bool, error)
	CreateAndCheckCharge(tokenID string, amount int) (bool, error)
	CreateTokenAndCharge1(donates []Donate) ([]string, []bool, error)
}

type paymentService struct{ repos utils.FileReader }

func NewPaymentService(repos utils.FileReader) PaymentService {
	return &paymentService{repos: repos}
}

func (ps *paymentService) CreatePaymentToken(donates []Donate) ([]string, error) {
	var tokenIDs []string
	cardData := url.Values{}

	year := 2026

	thisyear := time.Now().Year()
	// thismonth := time.Now().Month()

	for i, Donate := range donates {
		if year > thisyear {
			cardData = url.Values{
				"card[name]":             {Donate.Card.Name},
				"card[number]":           {Donate.CcNumber},
				"card[expiration_month]": {strconv.Itoa(Donate.Card.ExpirationMonth)},
				"card[expiration_year]":  {strconv.Itoa(Donate.Card.ExpirationYear)},
				"card[security_code]":    {Donate.CVV},
			}
			fmt.Println(i)

			client := &http.Client{}

			req, err := http.NewRequest("POST", "https://vault.omise.co/tokens", strings.NewReader(cardData.Encode()))
			if err != nil {
				return nil, fmt.Errorf("error creating POST request: %v", err)
			}

			req.SetBasicAuth("pkey_test_5z2kz7xvkilv3bw4kaa", "")

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error sending POST request for tokenization: %v\n", err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Unexpected status code for tokenization: %v\n", resp.StatusCode)
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response body: %v\n", err)
				continue
			}

			var tokenResponse TokenResponse
			err = json.Unmarshal(body, &tokenResponse)
			if err != nil {
				fmt.Printf("Error decoding token response body: %v\n", err)
				continue // Continue to the next donation
			}

			// Append token ID to the list
			tokenIDs = append(tokenIDs, tokenResponse.ID)
		}

		// Create HTTP client

	}

	return tokenIDs, nil
}

func CreateCharge(charge ChargeInfo) (string, error) {
	chargeData, err := json.Marshal(charge)
	if err != nil {
		return "", fmt.Errorf("error encoding charge data: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.omise.co/charges", bytes.NewBuffer(chargeData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %v", err)
	}
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	var chargeResponse ChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&chargeResponse); err != nil {
		return "", fmt.Errorf("error decoding response body: %v", err)
	}

	return chargeResponse.ID, nil
}

func (ps *paymentService) SortDonations() ([]Donate, error) {
	csv, err := ps.repos.Readfile()
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %v", err)
	}

	var donates []Donate
	lines := strings.Split(*csv, "\n")

	for i := range lines {
		if i == 0 {
			continue
		}

		line := strings.Split(lines[i], ",")
		if len(line) != 6 {
			return nil, fmt.Errorf("malformed input at line %d", i)
		}

		Amount, err := strconv.Atoi(line[1])
		if err != nil {
			return nil, fmt.Errorf("invalid Amount at line %d: %v", i, err)
		}

		month, err := strconv.Atoi(line[4])
		if err != nil {
			return nil, fmt.Errorf("invalid expiration month at line %d: %v", i, err)
		}

		year, err := strconv.Atoi(line[5])
		if err != nil {
			return nil, fmt.Errorf("invalid expiration year at line %d: %v", i, err)
		}

		// Check if the Card's expiration date is in the future
		expirationDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		if expirationDate.Before(time.Now()) {
			continue // Skip expired cards
		}

		Card := Card{
			Name:            line[0],
			ExpirationMonth: month,
			ExpirationYear:  year,
		}

		Donate := Donate{
			Amount:   Amount,
			CcNumber: line[2],
			CVV:      line[3],
			Card:     Card,
		}

		donates = append(donates, Donate)
	}

	return donates, nil
}

func (ps *paymentService) CalculateDonate(donates []Donate) (DonateInfo, error) {
	now := time.Now()

	var donateInfo DonateInfo

	for _, d := range donates {
		if d.Amount > donateInfo.TopDonate {
			donateInfo.TopDonate = d.Amount
			donateInfo.TopDonor = d.Card.Name
		}
		donateInfo.TotalCard++

		expirationDate := time.Date(d.Card.ExpirationYear, time.Month(d.Card.ExpirationMonth), 1, 0, 0, 0, 0, time.UTC)
		if expirationDate.Before(now) {
			donateInfo.InvalidSum += d.Amount
			donateInfo.InvalidCard++
		} else {
			donateInfo.ValidSum += d.Amount
			donateInfo.ValidCard++
		}
		donateInfo.TotalSum += d.Amount
	}

	return donateInfo, nil
}
func (ps *paymentService) CreateTokenAndCharge(donates []Donate) ([]string, []bool, error) {
	var tokenIDs []string
	var chargesPaid []bool

	// Create channels for token and charge responses
	tokenCh := make(chan string)
	chargeCh := make(chan bool)

	// WaitGroup to synchronize goroutines
	var wg sync.WaitGroup

	// Add the number of donations to the WaitGroup
	wg.Add(len(donates))

	// Process each donation in a separate goroutine
	for _, d := range donates {
		go func(d Donate) {
			defer wg.Done()

			// Create token
			tokenID, err := createToken(http.DefaultClient, d)
			if err != nil {
				fmt.Printf("Error creating token: %v\n", err)
				tokenCh <- "" // Send an empty string to indicate failure
				return
			}
			tokenCh <- tokenID

			// Create charge
			chargePaid, err := ps.CreateAndCheckCharge(tokenID, d.Amount)
			if err != nil {
				fmt.Printf("Error creating or checking charge: %v\n", err)
				chargeCh <- false // Send false to indicate failure
				return
			}
			chargeCh <- chargePaid
		}(d)
	}

	// Consume token IDs and charge status
	go func() {
		for tokenID := range tokenCh {
			if tokenID == "" {
				chargesPaid = append(chargesPaid, false)
				continue
			}
			tokenIDs = append(tokenIDs, tokenID)
		}
	}()

	go func() {
		for chargePaid := range chargeCh {
			chargesPaid = append(chargesPaid, chargePaid)
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Close channels
	close(tokenCh)
	close(chargeCh)

	return tokenIDs, chargesPaid, nil
}

func (ps *paymentService) CreateAndCheckCharge(tokenID string, amount int) (bool, error) {
	// Create charge
	chargeID, err := createCharge(tokenID, amount)
	if err != nil {
		return false, fmt.Errorf("error creating charge: %v", err)
	}

	// Check if charge was paid
	chargePaid, err := checkChargePaid(chargeID)
	if err != nil {
		return false, fmt.Errorf("error checking charge status: %v", err)
	}

	return chargePaid, nil
}

func (ps *paymentService) CreateTokenAndCharge1(donates []Donate) ([]string, []bool, error) {
	var tokenIDs []string
	var chargesPaid []bool

	client := &http.Client{}

	for _, Donate := range donates {
		// Create token
		tokenID, err := createToken(client, Donate)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating token: %v", err)
		}
		tokenIDs = append(tokenIDs, tokenID)

		// Create charge
		chargeID, err := createCharge(tokenID, Donate.Amount)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating charge: %v", err)
		}

		// Check if charge was paid
		chargePaid, err := checkChargePaid(chargeID)
		if err != nil {
			return nil, nil, fmt.Errorf("error checking charge status: %v", err)
		}
		chargesPaid = append(chargesPaid, chargePaid)
	}

	return tokenIDs, chargesPaid, nil
}

func createToken(client *http.Client, Donate Donate) (string, error) {
	cardData := url.Values{
		"Card[Name]":             {Donate.Card.Name},
		"Card[number]":           {Donate.CcNumber},
		"Card[expiration_month]": {strconv.Itoa(Donate.Card.ExpirationMonth)},
		"Card[expiration_year]":  {strconv.Itoa(Donate.Card.ExpirationYear)},
		"Card[security_code]":    {Donate.CVV},
	}

	req, err := http.NewRequest("POST", "https://vault.omise.co/tokens", strings.NewReader(cardData.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating POST request for tokenization: %v", err)
	}

	req.SetBasicAuth("pkey_test_5z2kz7xvkilv3bw4kaa", "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending POST request for tokenization: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code for tokenization: %v", resp.StatusCode)
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("error decoding token response body: %v", err)
	}

	return tokenResponse.ID, nil
}

func createCharge(tokenID string, Amount int) (string, error) {
	chargeInfo := ChargeInfo{
		Description: "Donation charge",
		Amount:      Amount,
		Currency:    "THB",
		Card:        tokenID,
	}

	chargeData, err := json.Marshal(chargeInfo)
	if err != nil {
		return "", fmt.Errorf("error encoding charge data: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.omise.co/charges", bytes.NewBuffer(chargeData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %v", err)
	}
	req.SetBasicAuth(secretKey, "")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	var chargeResponse ChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&chargeResponse); err != nil {
		return "", fmt.Errorf("error decoding response body: %v", err)
	}

	return chargeResponse.ID, nil
}

func checkChargePaid(chargeID string) (bool, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.omise.co/charges/%s", chargeID), nil)
	if err != nil {
		return false, fmt.Errorf("error creating HTTP request: %v", err)
	}
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error sending GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	var chargeResponse ChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&chargeResponse); err != nil {
		return false, fmt.Errorf("error decoding response body: %v", err)
	}

	return true, nil
}
