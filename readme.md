# ncore-leaderboard

This is a utility for scraping user ranks from nCore profiles and storing
the results in a SQLite database for sorting and analysis. It is designed
for efficient data collection using concurrent workers and low-level
parsing.

## Requirements

- Go 1.26 or later
- just (optional, for automated builds)
- sqlite3 (optional, for CLI queries)

## Installation

Clone the repository and install the necessary Go modules:

    git clone https://github.com/skidoodle/ncore-leaderboard
    cd ncore-leaderboard
    go mod tidy

## Usage

Create a `.env` file in the project root with your credentials:

    NICK=username
    PASS=cookie_pass

Build and execute the program using the provided justfile:

    just run

Alternatively, build the binary manually:

    go build -o ncore-leaderboard main.go
    ./ncore-leaderboard

The program processes the profile range defined in `main.go` and produces
a `leaderboard.db` file.

## Data Analysis

The results are stored in a SQLite database with indices on rank and ID.
You can query the data using any standard SQLite client or the following
commands:

    just top 10        # List the top 10 users by rank
    just query 1066    # Find the user at a specific rank

## Implementation Details

The scraper utilizes a worker pool to manage concurrent HTTP requests via
fasthttp. It performs raw byte signature scanning on HTML responses to 
extract data without the overhead of a DOM parser. Results are committed 
to SQLite in a single transaction to ensure atomicity and speed.

## License

This software is released under the GNU General Public License v3.0.
See the license file for details.
