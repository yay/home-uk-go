package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	entries := parseAndFilter()
	processAndSave(entries)
}

func parseAndFilter() []Entry {
	println("Parsing CSV file and filtering entries...")

	inputFile, err := os.Open("pp-complete.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer inputFile.Close()

	csvReader := csv.NewReader(inputFile)
	entries := make([]Entry, 0)

	for {
		columns, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		entry := toEntry(columns)

		// filtering entries
		if entry.postcode1 != "" &&
			entry.date.Year() >= 2015 &&
			entry.duration == Freehold &&
			sort.SearchStrings(LondonPostcodes, entry.postcode1) >= 0 {
			entries = append(entries, entry)
		}
	}

	println("Sorting entries...")

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].date.Year() < entries[j].date.Year()
	})

	return entries
}

type PostcodeYearTypeAgePricesMap = map[string]map[int]map[PropertyType]map[PropertyAge][]int

func processAndSave(entries []Entry) {
	println("Grouping entries...")

	// outputFile, err := os.Create("stats.json")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer outputFile.Close()

	// writer := bufio.NewWriter(outputFile)
	// count := len(entries)
	postcodeYearMap := make(PostcodeYearTypeAgePricesMap)
	for _, entry := range entries {
		yearTypeMap, ok := postcodeYearMap[entry.postcode1]
		if !ok {
			yearTypeMap = make(map[int]map[PropertyType]map[PropertyAge][]int)
			postcodeYearMap[entry.postcode1] = yearTypeMap
		}

		typeAgeMap, ok := yearTypeMap[entry.date.Year()]
		if !ok {
			typeAgeMap = make(map[PropertyType]map[PropertyAge][]int)
			yearTypeMap[entry.date.Year()] = typeAgeMap
		}

		agePricesMap, ok := typeAgeMap[entry.propertyType]
		if !ok {
			agePricesMap = make(map[PropertyAge][]int)
			typeAgeMap[entry.propertyType] = agePricesMap
		}

		prices, ok := agePricesMap[entry.propertyAge]
		if !ok {
			prices = make([]int, 0)
		}

		prices = append(prices, entry.price)
		agePricesMap[entry.propertyAge] = prices
	}

	// outputFile, err := os.Create("stats.json")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer outputFile.Close()

	// encoder := json.NewEncoder(outputFile)
	// encoder.Encode(postcodeYearMap)

	println("Generating JSON...")
	data, err := json.MarshalIndent(postcodeYearMap, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	println("Saving...")
	err = ioutil.WriteFile("stats.json", data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func minMax(slice []int) (int, int) {
	var max int = slice[0]
	var min int = slice[0]
	for _, value := range slice {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}

func findMedian(prices []int) float64 {
	count := len(prices)
	if count >= 2 && count%2 == 0 {
		middle := count / 2
		return float64(prices[middle-1]+prices[middle]) / 2
	}
	return float64(prices[count/2])
}

// ----------------------------------------------------------------

type PropertyType byte

const (
	Detached PropertyType = iota
	SemiDetached
	Terraced
	Flat
	Other
)

var propertyTypeNames = [...]string{"Detached", "SemiDetached", "Terraced", "Flat", "Other"}

func (t PropertyType) String() string {
	if int(t) >= len(propertyTypeNames) {
		return ""
	}
	return propertyTypeNames[t]
}

// MarshalText implements the encoding.TextMarshaler interface.
func (t PropertyType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// ----------------------------------------------------------------

type PropertyAge byte

const (
	New PropertyAge = iota
	Old
)

var propertyAgeNames = [...]string{"New", "Old"}

func (t PropertyAge) String() string {
	if int(t) >= len(propertyAgeNames) {
		return ""
	}
	return propertyAgeNames[t]
}

func (t PropertyAge) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// ----------------------------------------------------------------

type DurationOfTransfer byte

const (
	Freehold DurationOfTransfer = iota
	Leasehold
)

var durationOfTransferNames = [...]string{"Freehold", "Leasehold"}

func (t DurationOfTransfer) String() string {
	if int(t) >= len(durationOfTransferNames) {
		return ""
	}
	return durationOfTransferNames[t]
}

func (t DurationOfTransfer) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// ----------------------------------------------------------------

type Entry struct {
	price        int
	date         time.Time
	postcode1    string // postcodes can be reallocated and these changes are not reflected in the Price Paid Dataset
	postcode2    string
	propertyType PropertyType
	propertyAge  PropertyAge
	duration     DurationOfTransfer
}

func toEntry(columns []string) Entry {
	var entry Entry
	{
		i, err := strconv.Atoi(columns[1])
		if err != nil {
			log.Fatal(err)
		}
		entry.price = i
	}
	{
		time, err := time.Parse("2006-01-02 15:04", columns[2])
		if err != nil {
			log.Fatal(err)
		}
		entry.date = time
	}
	postcodeParts := strings.Split(columns[3], " ")
	entry.postcode1 = postcodeParts[0]
	if len(postcodeParts) == 2 {
		entry.postcode2 = postcodeParts[1]
	}
	entry.propertyType = toPropertyType(columns[4])
	entry.propertyAge = toPropertyAge(columns[5])
	entry.duration = toDurationOfTransfer(columns[6])

	return entry
}

func toPropertyType(s string) PropertyType {
	switch s {
	case "D":
		return Detached
	case "S":
		return SemiDetached
	case "T":
		return Terraced
	case "F":
		return Flat
	default:
		return Other // e.g. property comprises more than one large parcel of land
	}
}

func toPropertyAge(s string) PropertyAge {
	switch s {
	case "Y":
		return New
	default:
		return Old
	}
}

func toDurationOfTransfer(s string) DurationOfTransfer {
	switch s {
	case "F":
		return Freehold
	default:
		return Leasehold // leases of 7 years or less are not recorded in Price Paid Dataset
	}
}

var LondonPostcodes = []string{
	"EC1A", "EC1M", "EC1N", "EC1P", "EC1R", "EC1V", "EC1Y", "EC2A", "EC2M", "EC2N", "EC2P", "EC2R",
	"EC2V", "EC2Y", "EC3A", "EC3M", "EC3N", "EC3P", "EC3R", "EC3V", "EC4A", "EC4M", "EC4N", "EC4P",
	"EC4R", "EC4V", "EC4Y", "WC1A", "WC1B", "WC1E", "WC1H", "WC1N", "WC1R", "WC1V", "WC1X", "WC2A",
	"WC2B", "WC2E", "WC2H", "WC2N", "WC2R", "E1", "E2", "E3", "E4", "E5", "E6", "E7", "E8", "E9",
	"E10", "E11", "E12", "E13", "E14", "E15", "E16", "E17", "E18", "E19", "E20", "N1", "N2", "N3",
	"N4", "N5", "N6", "N7", "N8", "N9", "N10", "N11", "N12", "N13", "N14", "N15", "N16", "N17",
	"N18", "N19", "N20", "N21", "N22", "NW1", "NW2", "NW3", "NW4", "NW5", "NW6", "NW7", "NW8",
	"NW9", "NW10", "NW11", "SE1", "SE2", "SE3", "SE4", "SE5", "SE6", "SE7", "SE8", "SE9", "SE10",
	"SE11", "SE12", "SE13", "SE14", "SE15", "SE16", "SE17", "SE18", "SE19", "SE20", "SE21", "SE22",
	"SE23", "SE24", "SE25", "SE26", "SE27", "SE28", "SW1", "SW2", "SW3", "SW4", "SW5", "SW6",
	"SW7", "SW8", "SW9", "SW10", "SW11", "SW12", "SW13", "SW14", "SW15", "SW16", "SW17", "SW18",
	"SW19", "SW20", "W1", "W2", "W3", "W4", "W5", "W6", "W7", "W8", "W9", "W10", "W11", "W12",
	"W13", "W14",
}
