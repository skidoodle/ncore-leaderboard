package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	writeBatch   = 100
)

var (
	nick      string
	pass      string
	client    *http.Client
	wg        sync.WaitGroup
	mu        sync.Mutex
	lines     []Line
	processed int32
)

type Line struct {
	URL       string
	SecondCol int
}

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
					rankInt, err := strconv.Atoi(rank)
					if err != nil {
						log.Printf("Skipping profile %d due to invalid rank: %s\n", id, rank)
						return
					}
					mu.Lock()
					lines = append(lines, Line{URL: url, SecondCol: rankInt})
					mu.Unlock()
					atomic.AddInt32(&processed, 1)
					if atomic.LoadInt32(&processed)%writeBatch == 0 {
						writeSortedOutput()
					}
					printProgress()
				}
			}
		})
	})
}

func printProgress() {
	fmt.Printf("\rProcessed %d profiles...", atomic.LoadInt32(&processed))
}

func quicksort(lines []Line, low, high int) {
	if low < high {
		p := partition(lines, low, high)
		quicksort(lines, low, p-1)
		quicksort(lines, p+1, high)
	}
}

func partition(lines []Line, low, high int) int {
	pivot := lines[high].SecondCol
	i := low - 1
	for j := low; j < high; j++ {
		if lines[j].SecondCol < pivot {
			i++
			lines[i], lines[j] = lines[j], lines[i]
		}
	}
	lines[i+1], lines[high] = lines[high], lines[i+1]
	return i + 1
}

func sortLinesQuick() {
	if len(lines) > 1 {
		quicksort(lines, 0, len(lines)-1)
	}
}

func writeSortedOutput() {
	mu.Lock()
	defer mu.Unlock()

	sortLinesQuick()

	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Error creating output file: %v\n", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, line := range lines {
		if err := writer.Write([]string{line.URL, strconv.Itoa(line.SecondCol)}); err != nil {
			log.Printf("Error writing line to output file: %v\n", err)
		}
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

	writeSortedOutput()

	elapsedTime := time.Since(startTime)
	fmt.Printf("\nScraping and sorting completed in %s\n", elapsedTime)
}
