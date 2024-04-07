package service

import (
	"encoding/json"
	"fmt"
	"github.com/lichensio/api_server/db/model"
	repo "github.com/lichensio/api_server/db/repo"
	util "github.com/lichensio/api_server/internal/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type EmployeeService struct {
	repo repo.Repository
}

func NewEmployeeService(repo repo.Repository) *EmployeeService {
	return &EmployeeService{
		repo: repo,
	}
}

// LoadEmployeesFromInput assumes input is already a Go struct
// LoadEmployeesFromInput modified to use the helper function.
func (s *EmployeeService) LoadEmployeesFromInput(input []model.EmployeeInput) error {
	for _, empInput := range input {
		startDate, err := time.Parse("2006-01-02", empInput.StartDate)
		if err != nil {
			return err // Consider logging or handling the error as needed
		}

		// Load the employee, assuming LoadEmployees returns the ID of the loaded employee
		employee := &model.Employee{
			Name:      empInput.Name,
			StartDate: startDate,
		}
		err = s.repo.LoadEmployees([]*model.Employee{employee})
		if err != nil {
			return err // Consider logging or handling the error as needed
		}
		// fmt.Printf("Loaded employee ID: %d\n", employee.ID)

		// Assuming we now have employee.ID available
		// Iterate over each week's schedule and load schedules
		for weekType, weeklySchedule := range empInput.Weeks {
			err = s.loadWeeklySchedules(employee.ID, weekType, weeklySchedule)
			if err != nil {
				return err // Consider logging or handling the error as needed
			}
		}
	}
	return nil
}
func (s *EmployeeService) loadWeeklySchedules(employeeID uint, weekType string, weeklySchedule model.WeeklyScheduleInput) error {
	days := map[string][]model.ScheduleInput{
		"Monday":    weeklySchedule.Monday,
		"Tuesday":   weeklySchedule.Tuesday,
		"Wednesday": weeklySchedule.Wednesday,
		"Thursday":  weeklySchedule.Thursday,
		"Friday":    weeklySchedule.Friday,
		"Saturday":  weeklySchedule.Saturday,
		"Sunday":    weeklySchedule.Sunday,
	}

	for dayName, schedules := range days {
		for _, schedule := range schedules {
			startTime, err := time.Parse("15:04", schedule.Start)
			if err != nil {
				return err // Consider logging or handling the error as needed
			}
			endTime, err := time.Parse("15:04", schedule.End)
			if err != nil {
				return err // Consider logging or handling the error as needed
			}

			err = s.repo.UpdateSchedule(model.Schedule{
				EmployeeID: employeeID,
				WeekType:   weekType,
				DayName:    dayName,
				StartTime:  model.CustomTime{Time: startTime},
				EndTime:    model.CustomTime{Time: endTime},
			})
			if err != nil {
				return err // Consider logging or handling the error as needed
			}
		}
	}

	return nil
}
func (s *EmployeeService) FetchEmployeeSchedule(employeeID uint, month string, year int) ([]model.MonthlySchedule, error) {
	monthNum := util.MonthStringToNumber(month)

	if monthNum == 0 {
		return nil, fmt.Errorf("invalid month: %s", month)
	}

	// Fetch holidays for the month and year
	holidays, err := s.GetHolidaysForMonthYear(year, time.Month(monthNum))
	if err != nil {
		// Decide how to handle errors: log, return an error, or proceed without holidays
		log.Printf("Could not fetch holidays for %d-%02d: %v", year, monthNum, err)
		// Optional: return nil, err
	}

	// Convert holidays into a map for easy lookup
	holidayMap := make(map[string]string)
	for _, holiday := range holidays {
		holidayMap[holiday.HolidayDate.Format("2006-01-02")] = holiday.HolidayName
	}

	employee, err := s.repo.GetEmployeeWithSchedules(employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get start date for employee ID %d: %v", employeeID, err)
	}

	firstDayOfMonth := time.Date(year, time.Month(monthNum), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	entries := make([]model.MonthlySchedule, 0)
	for d := firstDayOfMonth; !d.After(lastDayOfMonth); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		weekType := util.WeekTypeForDate(employee.StartDate, d)
		var timeSlots []model.TimeSlot
		for _, sched := range employee.Schedules {
			if sched.WeekType == weekType && sched.DayName == d.Weekday().String() {
				formattedStartTime := sched.StartTime.Format("15:04")
				formattedEndTime := sched.EndTime.Format("15:04")

				timeSlots = append(timeSlots, model.TimeSlot{
					Start: formattedStartTime,
					End:   formattedEndTime,
				})
			}
		}

		holidayName := ""
		if name, ok := holidayMap[dateStr]; ok {
			holidayName = name
		}

		entries = append(entries, model.MonthlySchedule{
			Date:        dateStr,
			DayName:     d.Weekday().String(),
			HolidayName: holidayName,
			TimeSlots:   timeSlots,
		})
	}

	return entries, nil
}

func (s *EmployeeService) CalculateMonthlyHours(entries []model.MonthlySchedule) (float64, error) {
	var totalHours float64
	for _, entry := range entries {
		for _, slot := range entry.TimeSlots {
			hours, err := util.CalculateHours(slot.Start, slot.End)
			if err != nil {
				return 0, err // Handle the error appropriately
			}
			totalHours += hours
		}
	}
	return totalHours, nil
}

func (s *EmployeeService) DBCreate() error {
	return s.repo.DBCreate()
}

func (svc *EmployeeService) DBDelete() error {
	return svc.repo.DBDelete()
}

func (svc *EmployeeService) FetchAllEmployees() ([]model.Employee, error) {
	return svc.repo.GetEmployees()
}

type WeekSchedule struct {
	WeekType string          `json:"weekType"`
	Days     []DailySchedule `json:"days"`
}

type DailySchedule struct {
	DayName   string     `json:"dayName"`
	TimeSlots []TimeSlot `json:"timeSlots"`
}

type TimeSlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func (svc *EmployeeService) FetchEmployeeFormattedABWeek(employeeID uint) ([]WeekSchedule, error) {
	weekSchedules := []WeekSchedule{
		{WeekType: "A", Days: make([]DailySchedule, 7)},
		{WeekType: "B", Days: make([]DailySchedule, 7)},
	}

	// Define a fixed order and empty structure for the days of the week
	daysOrder := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for i, day := range daysOrder {
		weekSchedules[0].Days[i] = DailySchedule{DayName: day, TimeSlots: []TimeSlot{}}
		weekSchedules[1].Days[i] = DailySchedule{DayName: day, TimeSlots: []TimeSlot{}}
	}

	// Populate time slots for each week type
	for weekIndex, weekSchedule := range weekSchedules {
		schedules, err := svc.repo.GetSchedule(employeeID, weekSchedule.WeekType)
		if err != nil {
			return nil, err
		}

		for _, schedule := range schedules {
			dayIndex := findDayIndex(schedule.DayName, daysOrder)
			if dayIndex != -1 {
				startFormatted := schedule.StartTime.Format("15:04")
				endFormatted := schedule.EndTime.Format("15:04")
				weekSchedules[weekIndex].Days[dayIndex].TimeSlots = append(weekSchedules[weekIndex].Days[dayIndex].TimeSlots, TimeSlot{Start: startFormatted, End: endFormatted})
			}
		}
	}

	return weekSchedules, nil
}

func findDayIndex(dayName string, daysOrder []string) int {
	for i, day := range daysOrder {
		if day == dayName {
			return i
		}
	}
	return -1
}

// GetHolidaysForMonthYear tries to get holidays from the DB, fetches from the API if not found, and stores them
func (hs *EmployeeService) GetHolidaysForMonthYear(year int, month time.Month) ([]model.Holiday, error) {
	holidays, err := hs.repo.HolidayFindByMonthAndYear(year, month)
	if err != nil {
		return nil, err
	}

	// If holidays are not found in the database for the given month/year, fetch from API
	if len(holidays) == 0 {
		allHolidays, err := FetchHolidaysFromAPI(year)
		if err != nil {
			return nil, err
		}

		for dateStr, name := range allHolidays {
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue // skip if the date format is incorrect
			}

			// If the month matches the requested month, add to the database
			if date.Year() == year && date.Month() == month {
				holiday := model.Holiday{HolidayDate: date, HolidayName: name}
				err := hs.repo.HolidayCreate(&holiday)
				if err != nil {
					return nil, err
				}
				holidays = append(holidays, holiday)
			}
		}
	}

	return holidays, nil
}

// FetchHolidaysFromAPI fetches holidays for a given year from the API
func FetchHolidaysFromAPI(year int) (map[string]string, error) {
	url := fmt.Sprintf("https://calendrier.api.gouv.fr/jours-feries/metropole/%d.json", year)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var holidays map[string]string
	err = json.Unmarshal(body, &holidays)
	if err != nil {
		return nil, err
	}

	return holidays, nil
}
