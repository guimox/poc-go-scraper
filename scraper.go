package main

import (
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

func getFormattedDate(date time.Time) string {
	return date.Format("02/01/06")
}

func mapMealType(htmlContent string) string {
	if strings.Contains(htmlContent, "CAFÉ DA MANHÃ") {
		return "breakfast"
	} else if strings.Contains(htmlContent, "ALMOÇO") {
		return "lunch"
	} else if strings.Contains(htmlContent, "JANTAR") {
		return "dinner"
	}
	return ""
}

func extractMealColly(cell *colly.HTMLElement) []Meal {
	var meals []Meal

	// Get the HTML content of the cell
	htmlContent, err := cell.DOM.Html()
	if err != nil {
		log.Printf("Error getting HTML content: %v", err)
		return nil
	}

	// Split the content by <br> tags
	contentParts := strings.Split(htmlContent, "<br/>")
	for _, part := range contentParts {
		// Create a new DOM element for each part to extract meal name and icons
		partDOM, err := goquery.NewDocumentFromReader(strings.NewReader(part))
		if err != nil {
			log.Printf("Error creating DOM from part: %v", err)
			continue
		}

		// Extract meal name and icons
		icons := []string{}
		partDOM.Find("img").Each(func(_ int, img *goquery.Selection) {
			src, exists := img.Attr("src")
			if exists {
				icons = append(icons, src)
			}
		})

		// Extract the meal name by removing all HTML tags
		name := strings.TrimSpace(partDOM.Text())

		// Append the meal if it has a name
		if name != "" {
			log.Printf("Meal parsed: %s", name)
			meals = append(meals, Meal{
				Name:  name,
				Icons: icons,
			})
		}
	}

	return meals
}

func scrape(dateToScrape time.Time) (Response, error) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36"),
	)
	c.SetRequestTimeout(15 * time.Second)

	// Enable debug mode
	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting: %s", r.URL.String())
	})

	c.OnResponse(func(r *colly.Response) {
		// Log the response status code and body
		log.Printf("Response received from URL: %s\nStatus Code: %d\nResponse Body: %s", r.Request.URL, r.StatusCode, string(r.Body))
	})

	c.OnError(func(r *colly.Response, err error) {
		// Enhanced error logging
		if r != nil {
			log.Printf("Request URL: %s failed with response: %v\nStatus Code: %d\nResponse Body: %s\nError: %v", r.Request.URL, r, r.StatusCode, string(r.Body), err)
		} else {
			log.Printf("Request failed with error: %v", err)
		}
	})

	formattedDate := getFormattedDate(dateToScrape)
	responsePayload := Response{
		Date:    formattedDate,
		ImgMenu: nil,
		RuName:  "JARDIM BOTÂNICO",
		RuUrl:   "https://pra.ufpr.br/ru/cardapio-ru-jardim-botanico/",
		RuCode:  "BOT",
		Served:  []string{"breakfast", "lunch", "dinner"},
		Meals:   make(map[string][]Meal),
	}

	var currentMealType string
	var mealOptions []Meal

	// Declare flags and variables to manage state
	var dateFound bool
	var tableFound bool

	log.Printf("Starting to scrape the page: %s", responsePayload.RuUrl)

	c.OnHTML("strong", func(e *colly.HTMLElement) {
		if !dateFound {
			dateText := strings.TrimSpace(e.Text)
			log.Printf("Found date: %s", dateText)

			if strings.Contains(dateText, formattedDate) {
				log.Printf("Matching date found: %s", formattedDate)
				dateFound = true
				tableFound = false // Reset tableFound to ensure we only get the first table after the date
			}
		}
	})

	c.OnHTML("figure.wp-block-table", func(e *colly.HTMLElement) {
		if dateFound && !tableFound {
			log.Println("Found the table after matching date.")
			tableFound = true
			e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
				row.ForEach("td", func(_ int, cell *colly.HTMLElement) {
					htmlContent := cell.Text

					log.Printf("Processing cell content: %s", htmlContent)

					if strings.Contains(htmlContent, "CAFÉ DA MANHÃ") ||
						strings.Contains(htmlContent, "ALMOÇO") ||
						strings.Contains(htmlContent, "JANTAR") {
						// End of previous meal type, start new one
						if len(mealOptions) > 0 {
							log.Printf("Saving meals for: %s", currentMealType)
							responsePayload.Meals[currentMealType] = mealOptions
							mealOptions = nil
						}
						currentMealType = mapMealType(htmlContent)
						log.Printf("Current meal type: %s", currentMealType)
					} else {
						// Extract meal items and icons
						meals := extractMealColly(cell)
						if meals != nil {
							log.Printf("Extracted %d meals", len(meals))
							mealOptions = append(mealOptions, meals...)
						}
					}
				})
			})
		}
	})

	c.OnScraped(func(r *colly.Response) {
		log.Println("Scraping completed.")
	})

	err := c.Visit("https://pra.ufpr.br/ru/cardapio-ru-jardim-botanico/")
	if err != nil {
		log.Printf("Error visiting page: %v", err)
		return Response{}, err
	}

	// Add remaining meals if any
	if len(mealOptions) > 0 {
		responsePayload.Meals[currentMealType] = mealOptions
	}

	log.Println("Successfully scraped and created the response data.")
	return responsePayload, nil
}
