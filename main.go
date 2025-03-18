package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The base URL for the OData API
var URL = "https://opendata.riga.lv/odata/service/DeclaredPersons"

// Global HTTP client that will be used for all requests
var client *http.Client

// DeclaredPersons represents the entire response structure
// It contains a list of DeclaredPerson objects
type DeclaredPersons struct {
	// The API will return an array of person records
	// Note: We may need to adjust this field name based on the actual API response
	Value []DeclaredPerson `json:"value"`
}

// DeclaredPerson represents a single record in the API response
// The `json:"field_name"` tags tell Go how to map JSON fields to struct fields
type DeclaredPerson struct {
	ID          int    `json:"id"`
	Year        int    `json:"year"`
	Month       int    `json:"month"`
	Day         int    `json:"day"`
	Value       string `json:"value"`
	DistrictID  int    `json:"district_id"`
	DistrictName string `json:"district_name"`
}

// Parameters stores all command-line options provided by the user
type Parameters struct {
	Source   string // API URL
	District int    // Required district ID
	Year     int    // Optional year filter
	Month    int    // Optional month filter
	Day      int    // Optional day filter
	Limit    int    // Max number of records to retrieve
	Group    string // Grouping option (y, m, d, ym, yd, md)
	Out      string // Output JSON filename
}

// GroupedData represents records grouped by year, month, day or combinations
type GroupedData struct {
	GroupKey     string           // Key for the group (e.g., "2019" for year)
	Records      []DeclaredPerson // Records in this group
	Value        int              // Sum of values in this group
	Change       int              // Change from previous group
	Max          int              // Maximum value
	Min          int              // Minimum value
	Average      int              // Average value
	MaxDrop      int              // Maximum decrease
	MaxIncrease  int              // Maximum increase
}

// Add a new struct for the JSON output format
type OutputRecord struct {
	DistrictName string `json:"district_name"`
	Year         int    `json:"year,omitempty"`
	Month        int    `json:"month,omitempty"`
	Day          int    `json:"day,omitempty"`
	Value        int    `json:"value"`
	Change       int    `json:"change"`
	Max          int    `json:"Max"`
	Min          int    `json:"Min"`
	Average      int    `json:"Average"`
	MaxDrop      int    `json:"Max_drop"`
	MaxIncrease  int    `json:"Max_increase"`
}

// GetJSON makes an HTTP GET request and parses the JSON response
// It takes a URL and a pointer to a struct where the response will be stored
func GetJSON(url string, data interface{}) error {
	// Print the URL for debugging
	fmt.Println("Requesting data from:", url)
	
	// Make the HTTP GET request
	resp, err := http.Get(url)
	
	// Check if the request failed
	if err != nil {
		fmt.Println("HTTP request failed:", err)
		return err
	}
	
	// Make sure we close the response body when the function exits
	// defer means "do this at the end of the function"
	defer resp.Body.Close()
	
	// Read the entire response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response:", err)
		return err
	}
	
	// Show a preview of the response
	preview := string(bodyBytes)
	if len(preview) > 200 {
		preview = preview[:200] + "... (truncated)"
	}
	fmt.Println("Response status:", resp.Status)
	fmt.Println("Response preview:", preview)
	
	// Parse the JSON response into the provided data structure
	err = json.Unmarshal(bodyBytes, data)
	if err != nil {
		fmt.Println("Failed to parse JSON:", err)
	}
	return err
}

// groupData organizes records based on the grouping option and calculates statistics
func groupData(records []DeclaredPerson, groupBy string) map[string]GroupedData {
	// Create a map to hold our grouped data
	grouped := make(map[string]GroupedData)
	
	// First, group the records and calculate basic stats
	for _, record := range records {
		// Convert value string to int
		recordValue, _ := strconv.Atoi(record.Value)
		
		// Generate the key based on grouping option
		var key string
		switch groupBy {
		case "y":
			key = fmt.Sprintf("%d", record.Year)
		case "m":
			key = fmt.Sprintf("%d", record.Month)
		case "d":
			key = fmt.Sprintf("%d", record.Day)
		case "ym":
			key = fmt.Sprintf("%d-%02d", record.Year, record.Month)
		case "yd":
			key = fmt.Sprintf("%d-%02d", record.Year, record.Day)
		case "md":
			key = fmt.Sprintf("%02d-%02d", record.Month, record.Day)
		default:
			// If no grouping or invalid grouping, use a constant key
			key = "all"
		}
		
		// Get the existing group or create a new one
		group, exists := grouped[key]
		if !exists {
			group = GroupedData{
				GroupKey: key,
				Records:  []DeclaredPerson{},
				Value:    0,
				Min:      -1,  // Will be set with first record
				Max:      -1,  // Will be set with first record
			}
		}
		
		// Add this record to the group
		group.Records = append(group.Records, record)
		
		// Add the value to the group total
		group.Value += recordValue
		
		// Update Min and Max values
		if group.Min == -1 || recordValue < group.Min {
			group.Min = recordValue
		}
		if group.Max == -1 || recordValue > group.Max {
			group.Max = recordValue
		}
		
		// Update the group in our map
		grouped[key] = group
	}
	
	// Calculate averages for each group
	for key, group := range grouped {
		if len(group.Records) > 0 {
			group.Average = group.Value / len(group.Records)
			grouped[key] = group
		}
	}
	
	// Calculate max increase/drop within each group
	for key, group := range grouped {
		maxIncrease := 0
		maxDrop := 0
		
		// Sort records by year, month, day
		sort.Slice(group.Records, func(i, j int) bool {
			if group.Records[i].Year != group.Records[j].Year {
				return group.Records[i].Year < group.Records[j].Year
			}
			if group.Records[i].Month != group.Records[j].Month {
				return group.Records[i].Month < group.Records[j].Month
			}
			return group.Records[i].Day < group.Records[j].Day
		})
		
		// Calculate max increase/drop between consecutive records
		for i := 1; i < len(group.Records); i++ {
			prevValue, _ := strconv.Atoi(group.Records[i-1].Value)
			currValue, _ := strconv.Atoi(group.Records[i].Value)
			
			change := currValue - prevValue
			
			if change > maxIncrease {
				maxIncrease = change
			}
			if change < maxDrop {
				maxDrop = change
			}
		}
		
		group.MaxIncrease = maxIncrease
		group.MaxDrop = maxDrop
		grouped[key] = group
	}
	
	// Calculate change between groups (this needs ordered groups)
	// Convert to sorted slice first
	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// Calculate changes between consecutive groups
	for i := 1; i < len(keys); i++ {
		currGroup := grouped[keys[i]]
		prevGroup := grouped[keys[i-1]]
		
		currGroup.Change = currGroup.Value - prevGroup.Value
		grouped[keys[i]] = currGroup
	}
	
	return grouped
}

// fetchDeclairedPersons fetches data from the API based on the parameters
func fetchDeclairedPersons(params Parameters, client *http.Client) {
	// Parse the base URL
	baseURL, err := url.Parse(URL)
	if err != nil {
		fmt.Println("Error parsing base URL:", err)
		return
	}

	// Query parameters
	queryParams := url.Values{}

	// Build $filter clause
	filterClauses := []string{}

	if params.District > 0 {
		filterClauses = append(filterClauses, fmt.Sprintf("district_id eq %d", params.District))
	}
	if params.Year > 0 {
		filterClauses = append(filterClauses, fmt.Sprintf("year eq %d", params.Year))
	}
	if params.Month > 0 {
		filterClauses = append(filterClauses, fmt.Sprintf("month eq %d", params.Month))
	}
	if params.Day > 0 {
		filterClauses = append(filterClauses, fmt.Sprintf("day eq %d", params.Day))
	}

	if len(filterClauses) > 0 {
		queryParams.Add("$filter", strings.Join(filterClauses, " and "))
	}

	// Add the $top parameter
	queryParams.Add("$top", strconv.Itoa(params.Limit))

	// Encode parameters and append to URL
	baseURL.RawQuery = queryParams.Encode()

	finalURL := baseURL.String()
	fmt.Println("Using URL:", finalURL)
	
	// Get the data
	var persons DeclaredPersons
	err = GetJSON(finalURL, &persons)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d matching records\n", len(persons.Value))
	
	// Group the data if a grouping option was specified
	if params.Group != "" {
		// Group the filtered records
		groupedData := groupData(persons.Value, params.Group)
		
		// Print the results
		fmt.Printf("Found %d groups based on '%s' grouping\n", 
			len(groupedData), params.Group)
		
		// Convert map to slice for easier sorting
		groups := make([]GroupedData, 0, len(groupedData))
		for _, group := range groupedData {
			groups = append(groups, group)
		}
		
		// Sort groups by GroupKey
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].GroupKey < groups[j].GroupKey
		})
		
		// Prepare output records
		var outputRecords []OutputRecord
		
		for _, group := range groups {
			// Create a new output record
			record := OutputRecord{
				DistrictName: persons.Value[0].DistrictName,
				Value:        group.Value,
				Change:       group.Change,
				Max:          group.Max,
				Min:          group.Min,
				Average:      group.Average,
				MaxDrop:      group.MaxDrop,
				MaxIncrease:  group.MaxIncrease,
			}
			
			// Parse the group key to get year, month, day components
			if strings.Contains(params.Group, "y") {
				// Extract year from group key based on format
				if strings.HasPrefix(group.GroupKey, "20") { // Check if starts with year
					year, _ := strconv.Atoi(group.GroupKey[:4])
					record.Year = year
				}
			}
			
			if strings.Contains(params.Group, "m") {
				// Extract month from group key based on format
				if len(group.GroupKey) >= 7 && group.GroupKey[4] == '-' {
					// Format: YYYY-MM (ym grouping)
					month, _ := strconv.Atoi(group.GroupKey[5:7])
					record.Month = month
				} else if len(group.GroupKey) <= 2 || (len(group.GroupKey) >= 5 && group.GroupKey[2] == '-') {
					// Format: MM or MM-DD (m or md grouping)
					month, _ := strconv.Atoi(strings.Split(group.GroupKey, "-")[0])
					record.Month = month
				}
			}
			
			if strings.Contains(params.Group, "d") {
				// Extract day from group key based on format
				if strings.Contains(group.GroupKey, "-") {
					day, _ := strconv.Atoi(strings.Split(group.GroupKey, "-")[1])
					record.Day = day
				} else {
					day, _ := strconv.Atoi(group.GroupKey)
					record.Day = day
				}
			}
			
			outputRecords = append(outputRecords, record)
		}
		
		// Display the data to console
		for _, group := range groups {
			fmt.Printf("\nGroup: %s\n", group.GroupKey)
			fmt.Printf("  Records: %d\n", len(group.Records))
			fmt.Printf("  Value: %d\n", group.Value)
			fmt.Printf("  Change: %d\n", group.Change)
			fmt.Printf("  Min: %d\n", group.Min)
			fmt.Printf("  Max: %d\n", group.Max)
			fmt.Printf("  Average: %d\n", group.Average)
			fmt.Printf("  Max Drop: %d\n", group.MaxDrop)
			fmt.Printf("  Max Increase: %d\n", group.MaxIncrease)
		}
		
		// Save to JSON file if out parameter is specified
		if params.Out != "" {
			err := saveToJSON(params.Out, outputRecords)
			if err != nil {
				fmt.Printf("Error saving to JSON: %v\n", err)
			}
		}
	} else {
		// Display individual records (limit to 100)
		displayLimit := 100
		if displayLimit > len(persons.Value) {
			displayLimit = len(persons.Value)
		}
		
		for i := 0; i < displayLimit; i++ {
			person := persons.Value[i]
			fmt.Printf("ID: %d, District: %s (ID: %d), Year: %d, Month: %d, Day: %d, Value: %s\n",
				person.ID, person.DistrictName, person.DistrictID, 
				person.Year, person.Month, person.Day, person.Value)
		}
	}
}

// Add this function to save data to a JSON file
func saveToJSON(filename string, data []OutputRecord) error {
	// Create pretty JSON with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to file using os.WriteFile instead of ioutil.WriteFile
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return err
	}
	
	fmt.Printf("Data successfully exported to %s\n", filename)
	return nil
}

func main() {
	// Define command-line flags
	params := Parameters{}
	
	flag.StringVar(&params.Source, "source", URL, "Service address")
	flag.IntVar(&params.District, "district", 0, "District identifier (required)")
	flag.IntVar(&params.Year, "year", 0, "Year to filter data")
	flag.IntVar(&params.Month, "month", 0, "Month to filter data")
	flag.IntVar(&params.Day, "day", 0, "Day to filter data")
	flag.IntVar(&params.Limit, "limit", 100, "Maximum number of records to retrieve")
	flag.StringVar(&params.Group, "group", "", "Grouping option: y, m, d, ym, yd, md")
	flag.StringVar(&params.Out, "out", "", "Output file name for JSON export")
	
	// Parse command-line arguments
	flag.Parse()
	
	// TODO: add validation for other params
	
	// Validate required parameters
	if params.District == 0 {
		fmt.Println("Error: district parameter is required")
		flag.Usage()
		return
	}
	
	// Initialize HTTP client
	client := &http.Client{Timeout: time.Second * 10}
	
	// Get and process data
	fetchDeclairedPersons(params, client)

	// TODO: add better error handling
	// TODO: make district name lookup table

	/*
	Old code - keeping for reference
	old_url := fmt.Sprintf("http://api.example.com/%d", params.District)
	if params.Year > 0 {
		old_url += fmt.Sprintf("&year=%d", params.Year)
	}
	*/
}