package main

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	_ "modernc.org/sqlite"
)

const (
	baseURL      = "https://ncore.pro/profile.php?id="
	startProfile = 1
	endProfile   = 1812000
	concurrency  = 1000
	dbFile       = "leaderboard.db"

	maxConns     = 2000
	readTimeout  = 10 * time.Second
	writeTimeout = 5 * time.Second
	maxRetries   = 2
)

type Profile struct {
	ID   uint32
	Rank uint32
}

var (
	nick      string
	pass      string
	processed atomic.Int64
	found     atomic.Int64
	errors    atomic.Int64
	authFail  atomic.Bool
	client    *fasthttp.Client
)

func init() {
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load()
	nick = os.Getenv("NICK")
	pass = os.Getenv("PASS")

	client = &fasthttp.Client{
		Name:                     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		MaxConnsPerHost:          maxConns,
		MaxIdleConnDuration:      90 * time.Second,
		ReadTimeout:              readTimeout,
		WriteTimeout:             writeTimeout,
		NoDefaultUserAgentHeader: false,
		Dial: (&fasthttp.TCPDialer{
			Concurrency:      concurrency,
			DNSCacheDuration: 1 * time.Hour,
		}).Dial,
	}
}

func fastParseRank(data []byte) (uint32, bool) {
	labelMarker := []byte("<div class=\"profil_jobb_elso2\">Helyezés:</div>")
	idx := bytes.Index(data, labelMarker)
	if idx == -1 {
		return 0, false
	}

	searchArea := data[idx:]
	valueMarker := []byte("profil_jobb_masodik2\">")
	vIdx := bytes.Index(searchArea, valueMarker)
	if vIdx == -1 {
		return 0, false
	}

	start := vIdx + len(valueMarker)
	end := bytes.IndexByte(searchArea[start:], '<')
	if end == -1 {
		return 0, false
	}

	raw := searchArea[start : start+end]
	var val uint32
	hasDigits := false
	for _, b := range raw {
		if b >= '0' && b <= '9' {
			val = val*10 + uint32(b-'0')
			hasDigits = true
		}
	}

	return val, hasDigits
}

func worker(ctx context.Context, jobs <-chan uint32, results chan<- Profile, wg *sync.WaitGroup) {
	defer wg.Done()
	cookie := fmt.Sprintf("nick=%s; pass=%s", nick, pass)

	for id := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			processed.Add(1)
			var body []byte
			success := false

			for r := 0; r <= maxRetries; r++ {
				req := fasthttp.AcquireRequest()
				resp := fasthttp.AcquireResponse()
				req.SetRequestURI(fmt.Sprintf("%s%d", baseURL, id))
				req.Header.Set("Cookie", cookie)
				req.Header.Set("Accept-Encoding", "gzip")

				if err := client.Do(req, resp); err == nil && resp.StatusCode() == 200 {
					encoding := resp.Header.Peek("Content-Encoding")
					if bytes.Equal(encoding, []byte("gzip")) {
						body, _ = resp.BodyGunzip()
					} else {
						body = make([]byte, len(resp.Body()))
						copy(body, resp.Body())
					}

					if bytes.Contains(body, []byte("Belépés")) {
						authFail.Store(true)
						fasthttp.ReleaseRequest(req)
						fasthttp.ReleaseResponse(resp)
						break
					}

					if rank, ok := fastParseRank(body); ok {
						results <- Profile{ID: id, Rank: rank}
						found.Add(1)
					}
					success = true
					fasthttp.ReleaseRequest(req)
					fasthttp.ReleaseResponse(resp)
					break
				}
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
				if r < maxRetries {
					time.Sleep(time.Duration(r+1) * 100 * time.Millisecond)
				}
			}
			if !success {
				errors.Add(1)
			}
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if nick == "" || pass == "" {
		log.Fatal("NICK and PASS environment variables are required.")
	}

	total := endProfile - startProfile + 1
	fmt.Printf("Leaderboard: Workers=%d, Cores=%d, Range=[%d-%d]\n", concurrency, runtime.NumCPU(), startProfile, endProfile)

	jobs := make(chan uint32, concurrency*10)
	resultsChan := make(chan Profile, concurrency*10)
	var wg sync.WaitGroup

	results := make([]Profile, 0, 500000)
	collectorWg := sync.WaitGroup{}
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for p := range resultsChan {
			results = append(results, p)
		}
	}()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			worker(ctx, jobs, resultsChan, &wg)
		}()
	}

	startTime := time.Now()
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p := processed.Load()
				f := found.Load()
				e := errors.Load()
				if authFail.Load() {
					fmt.Print("\nWARNING: Auth failure/Rate limit detected!\n")
					authFail.Store(false)
				}
				if p == 0 {
					continue
				}
				elapsed := time.Since(startTime).Seconds()
				rps := float64(p) / elapsed
				eta := time.Duration(float64(int64(total)-p)/rps) * time.Second
				fmt.Printf("\rProgress: %d/%d (%.2f%%) | Found: %d | Err: %d | Speed: %.0f/s | ETA: %v    ",
					p, total, float64(p)/float64(total)*100, f, e, rps, eta.Round(time.Second))
			}
		}
	}()

	for i := uint32(startProfile); i <= uint32(endProfile); i++ {
		select {
		case jobs <- i:
		case <-ctx.Done():
			goto Done
		}
	}

Done:
	close(jobs)
	wg.Wait()
	close(resultsChan)
	collectorWg.Wait()

	fmt.Printf("\nScraping complete. Finalizing %d results...\n", len(results))

	slices.SortFunc(results, func(a, b Profile) int {
		return cmp.Compare(a.Rank, b.Rank)
	})

	if _, err := os.Stat(dbFile); err == nil {
		os.Remove(dbFile)
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	db.Exec("PRAGMA journal_mode = WAL")
	db.Exec("PRAGMA synchronous = OFF")
	db.Exec("PRAGMA cache_size = 1000000")
	db.Exec("CREATE TABLE leaderboard (rank INTEGER, id INTEGER)")

	fmt.Printf("Writing to %s...\n", dbFile)
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("INSERT INTO leaderboard(rank, id) VALUES(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, p := range results {
		_, err = stmt.Exec(p.Rank, p.ID)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Committing transaction...")
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Indexing ranks...")
	db.Exec("CREATE INDEX idx_rank ON leaderboard(rank)")
	db.Exec("CREATE INDEX idx_id ON leaderboard(id)")

	fmt.Println("\n--- TOP 5 LEADERBOARD ---")
	limit := 5
	if len(results) < 5 {
		limit = len(results)
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("#%-8d: ID %-8d (%s%d)\n", results[i].Rank, results[i].ID, baseURL, results[i].ID)
	}

	fmt.Printf("\nSuccess! Database saved in %v\n", time.Since(startTime).Round(time.Second))
}
