
**Golang Data Analysis Tool**

This guide explains how to build and use a Golang application to analyze data related to declared persons, with data up to 2018. The default district is Rīga (ID: 516).

**1. Data Acquisition**

* The application processes data with the following structure:
    * `ID`: Unique identifier
    * `District`: District ID (Defaults to 516 for Rīga)
    * `Year`: Year of data (up to 2018)
    * `Month`: Month of data
    * `Day`: Day of data
    * `Value`: Data value
    <!-- * `Limit`- does not work as intended -->

**2. Building the Executable**
* After downloading the main.go file locate it via terminal and execute appropriate commands based on the OS 

```bash
go build -o declared-persons-analyser main.go
```

* Compiles `main.go` to create the `declared-persons-analyser` executable.


* If you want platform specific executables you can target specific cross-platform builds:

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o declared-persons-analyser.exe main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o declared-persons-analyser main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o declared-persons-analyser main.go
```

**3. Running and Filtering**

```bash
# Basic usage Riga with the value of 516 is default one
./declared-persons-analyser -district 516 -out results.json

# Filter by year:
./declared-persons-analyser -district 516 -year 2017 -out yearly_data.json

# Filter by year and month:
./declared-persons-analyser -district 516 -year 2016 -month 6 -out june_2016.json
```

* The `-out` parameter specifies the JSON output file.
* Years must be 2018 or earlier.

**4. Filtering and Grouping Options**

* `-year`: Filters by year (up to 2018).
* `-month`: Filters by month (1-12).
* `-day`: Filters by day of month.
* `-group`: Groups data. Options: `y` (year), `m` (month), `d` (day), `ym` (year and month), `yd` (year and day), `md` (month and day).

**Grouping Examples**

```bash
# Group by year:
./declared-persons-analyser -district 516 -group y -out yearly_summary.json
```

