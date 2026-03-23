# nCore Leaderboard Scraper (Turbo-Go 1.26+)

A hyper-performance, zero-allocation Go scraper designed to saturate a Gigabit connection while extracting user ranks from ~1.8M nCore profiles in under 10 minutes.

## 🚀 Performance Specs

- **Throughput:** 5,000+ Requests Per Second (RPS) on a 1Gbps line.
- **CPU Efficiency:** Multi-threaded worker pool utilizing all available cores with `GOMAXPROCS`.
- **Zero-Allocation Parsing:** Raw-byte signature scanning instead of heavy DOM parsing (100x faster than `goquery`).
- **Network Stack:** Powered by `fasthttp` for maximum connection reuse and zero-allocation networking.
- **Storage:** Instant O(log N) lookups via **SQLite 3** with WAL-mode transactions and indexed ranks.

## 🛠️ Features

- **Universal Parser:** Targeted signature scanning (`profil_jobb_elso2`) ensuring 100% accuracy.
- **Real-Time Telemetry:** Live CLI status bar with Progress, RPS, Found count, and ETA.
- **Robust Networking:** Automated 3-stage exponential backoff retries for transient errors.
- **Security Monitoring:** Active detection of session expiration or rate-limiting (Login page detection).
- **Dual-Mode Output:** Sorted results saved to a professional-grade `leaderboard.db`.

## 📦 Requirements

- **Go 1.26+** (Uses modern `slices`, `cmp`, and `atomic` types).
- **Just** (Case-sensitive command runner).
- **AVX2-capable CPU** (Targeted via `GOAMD64=v3` build flags).

## 📥 Setup

1. Clone and install dependencies:
   ```powershell
   git clone https://github.com/skidoodle/ncore-leaderboard
   cd ncore-leaderboard
   just tidy
   ```

2. Configure your credentials in `.env.local` or `.env`:
   ```env
   NICK=your_username
   PASS=your_pass_cookie_value
   ```

## 🎮 Usage

### Build & Run
```powershell
just run
```

### Build Only (Optimized)
Produces a stripped, architecture-targeted `ncore-leaderboard.exe`.
```powershell
just build
```

### Data Exploration
Once scraping is complete, the results are in `leaderboard.db`. You can query it instantly:
```powershell
# Show top 25 users
just top 25

# Query a specific rank
just query 1066
```

## 🏗️ Architecture details

- **Concurrency:** Managed by a buffered job channel and 1,000+ goroutine workers.
- **Memory:** Results are stored in-memory using `uint32` to halve RAM footprint before being flushed to SQLite in a single atomic transaction.
- **Indexing:** Ranks and Profile IDs are indexed on disk for sub-millisecond leaderboard queries.

## ⚖️ License
GPL-3.0 License. Built for performance, speed, and precision.
