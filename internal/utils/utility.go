package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lichensio/api_server/db/model"
	"log"
	"sort"
	"time"
)

// monthStringToNumber converts month name to its numerical representation.
func MonthStringToNumber(month string) int {
	date, err := time.Parse("January", month)
	if err != nil {
		log.Printf("Error converting month to number: %v", err)
		return 1 // default to January on error
	}
	return int(date.Month())
}

// weekTypeForDate calculates whether the given date falls on Week A or Week B based on the employee's start date.
func WeekTypeForDate(startDate, currentDate time.Time) string {
	_, startWeek := startDate.ISOWeek()
	_, currentWeek := currentDate.ISOWeek()

	// Calculate the difference in weeks
	weeksSinceStart := currentWeek - startWeek

	// If the difference is negative, it means the currentDate is in a new year
	// Adjust weeksSinceStart accordingly by adding the number of weeks in a year
	// This simple adjustment assumes the dates are within a year of each other
	// For handling dates spanning multiple years, further adjustments are needed
	if weeksSinceStart < 0 {
		weeksSinceStart += 52 // Or 53, depending on the year
	}

	// Determine the week type based on the difference
	if weeksSinceStart%2 == 0 {
		return "A"
	}
	return "B"
}

// FormatSQLTime takes a SQL time string (in "15:04:05" format) and formats it to "HH:MM".
func FormatSQLTime(sqlTime string) string {
	t, err := time.Parse("15:04:05", sqlTime)
	if err != nil {
		log.Printf("Error parsing time: %v", err)
		return ""
	}
	return t.Format("15:04")
}

func CountSchedules(employees model.EmployeesInput) map[string]int {
	counts := make(map[string]int)
	for _, employee := range employees {
		count := 0 // Initialize count for each employee
		for _, week := range employee.Weeks {
			count += len(week.Monday)
			count += len(week.Tuesday)
			count += len(week.Wednesday)
			count += len(week.Thursday)
			count += len(week.Friday)
			count += len(week.Saturday)
			count += len(week.Sunday)
		}
		counts[employee.Name] = count // Assign the total count to the employee's name
	}
	return counts
}

// hashJSON returns a SHA-256 hash of the JSON object represented by jsonString.
// It normalizes the JSON object to ensure consistent hashing.
func HashJSON(jsonString string) (string, error) {
	var object interface{}

	// Unmarshal JSON into an interface.
	if err := json.Unmarshal([]byte(jsonString), &object); err != nil {
		return "", err
	}

	// Marshal the object back into JSON with consistent key ordering.
	normalizedJSON, err := json.Marshal(normalize(object))
	if err != nil {
		return "", err
	}

	// Compute the SHA-256 hash.
	hash := sha256.Sum256(normalizedJSON)
	return hex.EncodeToString(hash[:]), nil
}

// normalize recursively sorts the JSON object to ensure consistent key ordering.
func normalize(value interface{}) interface{} {
	switch value := value.(type) {
	case map[string]interface{}:
		// Create a sorted list of map keys.
		keys := make([]string, 0, len(value))
		for k := range value {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Build a new map with sorted keys.
		sortedMap := make(map[string]interface{}, len(keys))
		for _, k := range keys {
			sortedMap[k] = normalize(value[k])
		}
		return sortedMap
	case []interface{}:
		// Normalize each element of the array.
		for i, v := range value {
			value[i] = normalize(v)
		}
	}
	return value
}

func GetEmployeeIDByName(employees []model.Employee, name string) (uint, error) {
	for _, employee := range employees {
		if employee.Name == name {
			return employee.ID, nil
		}
	}
	return 0, errors.New("employee not found")
}

// Compares two slices of MonthlySchedule for equality.
// Returns true if they are the same; otherwise, returns false and a summary of differences.
func CompareMonthlySchedules(a, b []model.MonthlySchedule) (bool, string) {
	if len(a) != len(b) {
		return false, fmt.Sprintf("Schedules length mismatch: %d vs %d", len(a), len(b))
	}

	for i := range a {
		if a[i].Date != b[i].Date || a[i].DayName != b[i].DayName {
			return false, fmt.Sprintf("Mismatch at index %d: Different Date or DayName", i)
		}
		if !compareTimeSlots(a[i].TimeSlots, b[i].TimeSlots) {
			return false, fmt.Sprintf("Mismatch at index %d: Different TimeSlots", i)
		}
	}
	return true, "Schedules are identical"
}

// Compares two slices of TimeSlot for equality, allowing for more flexible matching.
func compareTimeSlots(a, b []model.TimeSlot) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Start != b[i].Start || a[i].End != b[i].End {
			// Debug output to identify the discrepancy
			fmt.Printf("Mismatch in TimeSlot at index %d: %+v vs. %+v\n", i, a[i], b[i])
			return false
		}
	}
	return true
}

func CalculateHours(start, end string) (float64, error) {
	layout := "15:04"
	startTime, err := time.Parse(layout, start)
	if err != nil {
		return 0, err
	}

	endTime, err := time.Parse(layout, end)
	if err != nil {
		return 0, err
	}

	if endTime.Before(startTime) {
		// This handles cases where the end time is past midnight, indicating the next day.
		// Adjust endTime by adding 24 hours to it.
		endTime = endTime.Add(24 * time.Hour)
	}

	duration := endTime.Sub(startTime)
	return duration.Hours(), nil
}

// Other utility functions...
