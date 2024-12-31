# nCore Profile Scraper

This is a Go program for scraping and sorting user profile data from [nCore](https://ncore.pro/), saving results to a CSV file.

## Key Features

- **Concurrent Scraping:** Fast, parallel processing of profiles.
- **Quicksort Algorithm:** Efficient sorting by attributes.
- **Batch Writing:** Saves data incrementally to reduce memory usage.

## Setup

1. Clone the repository and install dependencies:
   ```bash
   git clone https://github.com/skidoodle/scrapencore
   cd scrapencore
   go mod tidy
   ```

2. Create a .env file with your credentials:
    ```env
    NICK=your_username
    PASS=your_pass
    ```

## Usage
Run the scraper:
    ```bash
    go run main.go
    ```

- Scrapes profiles from the configured range.
- Outputs sorted data to output.log in CSV format.
  
## Configuration
Edit these parameters in `main.go` as needed:

`startProfile`, `endProfile`: Profile ID range.  
`concurrency`: Number of concurrent requests.  
`outputFile`: Output file name.  
`writeBatch`: Profiles processed per save.  

## Output Format
The CSV file `output.log` contains:

1. Profile URL
2. Attribute Value (e.g., rank)

## License
This project is licensed under the MIT License.



