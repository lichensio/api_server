package service

import (
	"fmt"
	"github.com/lichensio/api_server/db/model"
	repo "github.com/lichensio/api_server/db/repo"
	util "github.com/lichensio/api_server/internal/utils"
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

	employee, err := s.repo.GetEmployeeWithSchedules(employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get start date for employee ID %d: %v", employeeID, err)
	}
	// fmt.Println(employee)

	firstDayOfMonth := time.Date(year, time.Month(monthNum), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	entries := make([]model.MonthlySchedule, 0)
	for d := firstDayOfMonth; !d.After(lastDayOfMonth); d = d.AddDate(0, 0, 1) {
		weekType := util.WeekTypeForDate(employee.StartDate, d)
		// fmt.Println(weekType)
		var timeSlots []model.TimeSlot
		for _, sched := range employee.Schedules {
			// fmt.Println(sched.WeekType, weekType)
			// fmt.Println(sched.DayName, d.Weekday().String())
			if sched.WeekType == weekType && sched.DayName == d.Weekday().String() {

				formattedStartTime := sched.StartTime.Format("15:04")
				formattedEndTime := sched.EndTime.Format("15:04")

				timeSlots = append(timeSlots, model.TimeSlot{
					Start: formattedStartTime,
					End:   formattedEndTime,
				})
				// fmt.Println(timeSlots)
			}
		}

		entries = append(entries, model.MonthlySchedule{
			Date:      d.Format("2006-01-02"),
			DayName:   d.Weekday().String(),
			TimeSlots: timeSlots,
		})
	}

	return entries, nil
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
