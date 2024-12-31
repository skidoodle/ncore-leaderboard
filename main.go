package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
)

const (
	baseURL      = "https://ncore.pro/profile.php?id="
	startProfile = 1
	endProfile   = 1812000
	concurrency  = 50
	outputFile   = "output.log"
)

var (
	nick   string
	pass   string
	client *http.Client
	wg     sync.WaitGroup
	mu     sync.Mutex
)

func init() {
	_ = godotenv.Load(".env.local")
	godotenv.Load()
	nick = os.Getenv("NICK")
	pass = os.Getenv("PASS")
	client = &http.Client{}
}

func fetchProfile(id int) {
	defer wg.Done()
	url := fmt.Sprintf("%s%d", baseURL, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request for %d: %v\n", id, err)
		return
	}
	req.Header.Set("Cookie", fmt.Sprintf("nick=%s; pass=%s", nick, pass))

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching profile %d: %v\n", id, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Error parsing profile document for %d: %v\n", id, err)
		return
	}

	doc.Find(".userbox_tartalom_mini").Each(func(i int, s *goquery.Selection) {
		s.Find(".profil_jobb_elso2").Each(func(ii int, labelSel *goquery.Selection) {
			label := labelSel.Text()
			valueSel := labelSel.Next()
			if valueSel.Length() > 0 {
				value := valueSel.Text()
				switch label {
				case "Helyezés:":
					rank := strings.TrimSuffix(value, ".")
					mu.Lock()
					logToFile(url, rank)
					mu.Unlock()
					fmt.Printf("\rProcessed %d/%d profiles...", id, endProfile)
				}
			}
		})
	})
}

func logToFile(url string, rank string) {
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening output file: %v\n", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{url, rank}); err != nil {
		log.Fatalf("Error writing to output file: %v\n", err)
	}
}

func main() {
	if _, err := os.Stat(outputFile); err == nil {
		var response string
		fmt.Printf("Output file %s already exists. Overwrite? (yes/no): ", outputFile)
		fmt.Scanln(&response)
		if response != "yes" {
			log.Println("Exiting. Please rename or remove the existing output file.")
			return
		}
		err := os.Remove(outputFile)
		if err != nil {
			log.Fatalf("Failed to remove existing output file: %v\n", err)
		}
	}

	fmt.Println("Scraping in progress...")
	startTime := time.Now()

	for i := startProfile; i <= endProfile; i++ {
		wg.Add(1)
		go fetchProfile(i)

		if i%concurrency == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
	elapsedTime := time.Since(startTime)
	fmt.Printf("\nScraping completed in %s\n", elapsedTime)
}
