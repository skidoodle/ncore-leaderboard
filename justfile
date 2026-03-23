set shell := ["powershell", "-NoProfile", "-Command"]

# Build the most optimized executable for modern CPUs
build:
    $env:GOAMD64="v3"; go build -ldflags="-s -w" -trimpath -o ncore-leaderboard.exe main.go
    @echo "Build complete: ncore-leaderboard.exe"

# Run the scraper
run: build
    ./ncore-leaderboard.exe

# Cleanup all database and build artifacts
clean:
    if (Test-Path ncore-leaderboard.exe) { Remove-Item ncore-leaderboard.exe -ErrorAction SilentlyContinue }
    if (Test-Path leaderboard.db) { Remove-Item leaderboard.db -ErrorAction SilentlyContinue }
    if (Test-Path leaderboard.db-wal) { Remove-Item leaderboard.db-wal -ErrorAction SilentlyContinue }
    if (Test-Path leaderboard.db-shm) { Remove-Item leaderboard.db-shm -ErrorAction SilentlyContinue }
    @echo "Cleanup complete."

# Query the database (Example: just query 100)
query rank="1":
    sqlite3 leaderboard.db "SELECT * FROM leaderboard WHERE rank = {{ rank }} LIMIT 1;"

# Show top N from the database (Example: just top 5)
top n="10":
    sqlite3 leaderboard.db "SELECT * FROM leaderboard ORDER BY rank ASC LIMIT {{ n }};"

# Formatting
fmt:
    go fmt ./...

# Tidy dependencies
tidy:
    go mod tidy
