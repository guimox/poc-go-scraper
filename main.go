package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type Request struct {
	Date string `json:"date"`
}

type Response struct {
	Date    string            `json:"date"`
	ImgMenu *string           `json:"imgMenu"`
	RuName  string            `json:"ruName"`
	RuUrl   string            `json:"ruUrl"`
	RuCode  string            `json:"ruCode"`
	Served  []string          `json:"served"`
	Meals   map[string][]Meal `json:"meals"`
}

type Meal struct {
	Name  string   `json:"name"`
	Icons []string `json:"icons"`
}

func handler(ctx context.Context, request Request) (Response, error) {
	var dateToScrape time.Time
	var err error

	// Check if the date is provided; if not, use today's date
	if request.Date == "" {
		dateToScrape = time.Now()
		log.Printf("No date provided, using today's date: %s", dateToScrape.Format("2006-01-02"))
	} else {
		log.Printf("Received date: %s", request.Date)
		dateToScrape, err = time.Parse("2006-01-02", request.Date)
		if err != nil {
			log.Printf("Error parsing date: %v", err)
			return Response{}, err
		}
	}

	responseData, err := scrape(dateToScrape)
	if err != nil {
		log.Printf("Error scraping menu: %v", err)
		return Response{}, err
	}

	return responseData, nil
}

func main() {
	// Start the Lambda handler
	lambda.Start(handler)
}
