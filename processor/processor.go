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

type donate struct {
	amount   int
	ccnumber string
	cvv      string
	card     card
}

type card struct {
	name            string
	expirationMonth int
	expirationYear  int
}

type CardDetails struct {
	Name            string `json:"name"`
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
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	ReturnURI   string `json:"return_uri"`
	Card        string `json:"card"`
}

type ChargeResponse struct {
	ID string `json:"id"`
}

const secretKey = "skey_test_5z2kz7ytse0t03341hs"

type PaymentService interface {
	SortDonations() (donates []donate, err error)
	CalculateDonate(donates []donate) (DonateInfo DonateInfo, err error)
	CreatePaymentToken(donates []donate) ([]string, error)
}

type paymentService struct{ repos utils.FileReader }

func NewPaymentService(repos utils.FileReader) PaymentService {
	return &paymentService{repos: repos}
}

func (ps *paymentService) CreatePaymentToken(donates []donate) ([]string, error) {
	var tokenIDs []string
	cardData := url.Values{}

	year := 2026

	thisyear := time.Now().Year()
	// thismonth := time.Now().Month()

	for i, donate := range donates {
		if year > thisyear {
			cardData = url.Values{
				"card[name]":             {donate.card.name},
				"card[number]":           {donate.ccnumber},
				"card[expiration_month]": {strconv.Itoa(donate.card.expirationMonth)},
				"card[expiration_year]":  {strconv.Itoa(year)},
				"card[security_code]":    {donate.cvv},
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

func (ps *paymentService) SortDonations() ([]donate, error) {
	csv, err := ps.repos.Readfile()
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %v", err)
	}

	var donates []donate
	lines := strings.Split(*csv, "\n")

	for i := range lines {
		if i == 0 {
			continue
		}

		line := strings.Split(lines[i], ",")
		if len(line) != 6 {
			return nil, fmt.Errorf("malformed input at line %d", i)
		}

		amount, err := strconv.Atoi(line[1])
		if err != nil {
			return nil, fmt.Errorf("invalid amount at line %d: %v", i, err)
		}

		month, err := strconv.Atoi(line[4])
		if err != nil {
			return nil, fmt.Errorf("invalid expiration month at line %d: %v", i, err)
		}

		year, err := strconv.Atoi(line[5])
		if err != nil {
			return nil, fmt.Errorf("invalid expiration year at line %d: %v", i, err)
		}

		card := card{
			name:            line[0],
			expirationMonth: month,
			expirationYear:  year,
		}

		donate := donate{
			amount:   amount,
			ccnumber: line[2],
			cvv:      line[3],
			card:     card,
		}

		donates = append(donates, donate)
	}

	return donates, nil
}

func (ps *paymentService) CalculateDonate(donates []donate) (DonateInfo, error) {
	now := time.Now()

	var donateInfo DonateInfo

	for _, d := range donates {
		if d.amount > donateInfo.TopDonate {
			donateInfo.TopDonate = d.amount
			donateInfo.TopDonor = d.card.name
		}
		donateInfo.TotalCard++

		expirationDate := time.Date(d.card.expirationYear, time.Month(d.card.expirationMonth), 1, 0, 0, 0, 0, time.UTC)
		if expirationDate.Before(now) {
			donateInfo.InvalidSum += d.amount
			donateInfo.InvalidCard++
		} else {
			donateInfo.ValidSum += d.amount
			donateInfo.ValidCard++
		}
		donateInfo.TotalSum += d.amount
	}

	return donateInfo, nil
}
