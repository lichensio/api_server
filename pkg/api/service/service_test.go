package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/lichensio/api_server/db/model"
	repo "github.com/lichensio/api_server/db/repo"
	"github.com/lichensio/api_server/internal/utils"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"testing"
)

// setupTestDB initializes the test database, applies migrations, and returns a gorm.DB instance.
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	err := godotenv.Load(".env") // Adjust the path to your .env file
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Apply migrations
	err = db.AutoMigrate(&model.Employee{}, &model.Schedule{})
	require.NoError(t, err)

	// Cleanup function to be called after tests
	cleanup := func() {
		if err := db.Migrator().DropTable(&model.Schedule{}); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("Warning: Failed to clean up schedules table: %v", err)
			}
		}
		if err := db.Migrator().DropTable(&model.Employee{}); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("Warning: Failed to clean up employees table: %v", err)
			}
		}
	}

	return db, cleanup
}

// setupTestService initializes EmployeeService with a test database for use in tests.
func setupTestService(t *testing.T) (*EmployeeService, func()) {
	db, cleanup := setupTestDB(t)
	repository := repo.NewRepositoryWithDB(db) // Assumes NewRepository can accept *gorm.DB directly
	employeeService := NewEmployeeService(repository)
	return employeeService, cleanup
}

// Define your JSON input here as a raw string for testing or load it from a file
var jsonInput = `[

  {
      "name": "Delphine",
      "startDate": "2024-01-08",
      "weeks": {
        "A": {
          "Monday": [],
          "Tuesday": [{"start": "9:00", "end": "12:00"}, {"start": "13:00", "end": "17:45"}],
          "Wednesday": [{"start": "9:00", "end": "12:00"}, {"start": "13:00", "end": "18:45"}],
          "Thursday": [ {"start": "12:45", "end": "19:45"}],
          "Friday": [{"start": "13:00", "end": "20:00"}],
          "Saturday": [{"start": "13:00", "end": "20:00"}],
          "Sunday": []
        },
        "B": {
          "Monday": [ {"start": "12:45", "end": "19:45"}],
          "Tuesday": [ {"start": "11:45", "end": "19:45"}],
          "Wednesday": [{"start": "12:45", "end": "19:45"}],
          "Thursday": [],
          "Friday": [{"start": "9:00", "end": "12:00"}, {"start": "13:00", "end": "17:45"}],
          "Saturday": [{"start": "09:00", "end": "16:00"}],
          "Sunday": []
        }
      }
    },
    {
      "name": "Henny Honore",
      "startDate": "2024-02-24",
      "weeks": {
        "A": {
          "Monday": [{"start": "9:00", "end": "12:00"}, {"start": "13:00", "end": "17:00"}],
          "Tuesday": [],
          "Wednesday": [{"start": "10:00", "end": "13:00"}, {"start": "14:00", "end": "18:45"}],
          "Thursday": [{"start": "9:00", "end": "13:00"}, {"start": "15:00", "end": "19:00"}],
          "Friday": [{"start": "13:00", "end": "20:00"}],
          "Saturday": [{"start": "13:00", "end": "20:00"}],
          "Sunday": []
        },
        "B": {
          "Monday": [{"start": "10:00", "end": "13:00"}, {"start": "14:00", "end": "19:00"}],
          "Tuesday": [{"start": "11:45", "end": "19:45"}],
          "Wednesday": [{"start": "12:00", "end": "19:45"}],
          "Thursday": [],
          "Friday": [{"start": "9:00", "end": "13:00"}, {"start": "14:00", "end": "18:00"}],
          "Saturday": [{"start": "9:00", "end": "14:00"}],
          "Sunday": []
        }
      }
    }

]`

func TestLoadEmployeesFromInput(t *testing.T) {
	// Setup the service with a test database connection
	employeeService, cleanup := setupTestService(t)
	defer cleanup()
	var employees []model.EmployeeInput
	var appEmployees []model.Employee

	employeeService.repo.CleanupDatabase() // Assuming this properly cleans the test database
	// Unmarshal the JSON into the EmployeesInput slice
	if err := json.Unmarshal([]byte(jsonInput), &employees); err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}
	scheduleNumberPerEmploye := util.CountSchedules(employees)
	var tot int
	for _, count := range scheduleNumberPerEmploye {
		tot += count
	}
	fmt.Println(tot)

	// fmt.Println(employees)
	// Load employees and their schedules from the JSON input

	err := employeeService.LoadEmployeesFromInput(employees)
	require.NoError(t, err, "Failed to load employees and schedules from input")

	// Fetch all employees to verify the outcomes
	appEmployees, err = employeeService.repo.GetEmployees()
	require.NoError(t, err, "Failed to retrieve employees")

	// Check if the correct number of employees are loaded
	require.Equal(t, 2, len(appEmployees), "Expected number of employees does not match")

	// Further verification could involve checking the schedules for each employee.
	// This could include verifying the total number of schedules, specific schedule details, etc.
	for _, employee := range appEmployees {
		schedulesA, errA := employeeService.repo.GetEmployeeWithSchedulesByWeekType(employee.ID, "A")
		schedulesB, errB := employeeService.repo.GetEmployeeWithSchedulesByWeekType(employee.ID, "B")
		require.NoError(t, errA, "Failed to retrieve schedules A for employee")
		require.NoError(t, errB, "Failed to retrieve schedules B for employee")
		// Add assertions about the schedules here, such as checking the number of schedules matches expectations
		//fmt.Println(schedulesA.Schedules)
		//fmt.Println(schedulesB.Schedules)
		require.Equal(t, scheduleNumberPerEmploye[employee.Name], len(schedulesA.Schedules)+len(schedulesB.Schedules), "Expected number of schedule does not match")
	}
}

func TestFetchEmployeeSchedule(t *testing.T) {
	// Setup the service with a test database connection
	employeeService, cleanup := setupTestService(t)
	defer cleanup()
	schedulesResult := []model.MonthlySchedule{
		{Date: "2024-03-01", DayName: "Friday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "13:00"}, {Start: "14:00", End: "18:00"}}},
		{Date: "2024-03-02", DayName: "Saturday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "14:00"}}},
		{Date: "2024-03-03", DayName: "Sunday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-04", DayName: "Monday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "12:00"}, {Start: "13:00", End: "17:00"}}},
		{Date: "2024-03-05", DayName: "Tuesday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-06", DayName: "Wednesday", TimeSlots: []model.TimeSlot{{Start: "10:00", End: "13:00"}, {Start: "14:00", End: "18:45"}}},
		{Date: "2024-03-07", DayName: "Thursday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "13:00"}, {Start: "15:00", End: "19:00"}}},
		{Date: "2024-03-08", DayName: "Friday", TimeSlots: []model.TimeSlot{{Start: "13:00", End: "20:00"}}},
		{Date: "2024-03-09", DayName: "Saturday", TimeSlots: []model.TimeSlot{{Start: "13:00", End: "20:00"}}},
		{Date: "2024-03-10", DayName: "Sunday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-11", DayName: "Monday", TimeSlots: []model.TimeSlot{{Start: "10:00", End: "13:00"}, {Start: "14:00", End: "19:00"}}},
		{Date: "2024-03-12", DayName: "Tuesday", TimeSlots: []model.TimeSlot{{Start: "11:45", End: "19:45"}}},
		{Date: "2024-03-13", DayName: "Wednesday", TimeSlots: []model.TimeSlot{{Start: "12:00", End: "19:45"}}},
		{Date: "2024-03-14", DayName: "Thursday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-15", DayName: "Friday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "13:00"}, {Start: "14:00", End: "18:00"}}},
		{Date: "2024-03-16", DayName: "Saturday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "14:00"}}},
		{Date: "2024-03-17", DayName: "Sunday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-18", DayName: "Monday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "12:00"}, {Start: "13:00", End: "17:00"}}},
		{Date: "2024-03-19", DayName: "Tuesday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-20", DayName: "Wednesday", TimeSlots: []model.TimeSlot{{Start: "10:00", End: "13:00"}, {Start: "14:00", End: "18:45"}}},
		{Date: "2024-03-21", DayName: "Thursday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "13:00"}, {Start: "15:00", End: "19:00"}}},
		{Date: "2024-03-22", DayName: "Friday", TimeSlots: []model.TimeSlot{{Start: "13:00", End: "20:00"}}},
		{Date: "2024-03-23", DayName: "Saturday", TimeSlots: []model.TimeSlot{{Start: "13:00", End: "20:00"}}},
		{Date: "2024-03-24", DayName: "Sunday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-25", DayName: "Monday", TimeSlots: []model.TimeSlot{{Start: "10:00", End: "13:00"}, {Start: "14:00", End: "19:00"}}},
		{Date: "2024-03-26", DayName: "Tuesday", TimeSlots: []model.TimeSlot{{Start: "11:45", End: "19:45"}}},
		{Date: "2024-03-27", DayName: "Wednesday", TimeSlots: []model.TimeSlot{{Start: "12:00", End: "19:45"}}},
		{Date: "2024-03-28", DayName: "Thursday", TimeSlots: []model.TimeSlot{}},
		{Date: "2024-03-29", DayName: "Friday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "13:00"}, {Start: "14:00", End: "18:00"}}},
		{Date: "2024-03-30", DayName: "Saturday", TimeSlots: []model.TimeSlot{{Start: "09:00", End: "14:00"}}},
		{Date: "2024-03-31", DayName: "Sunday", TimeSlots: []model.TimeSlot{}},
	}

	var employees []model.EmployeeInput

	employeeService.repo.CleanupDatabase() // Assuming this properly cleans the test database
	// Unmarshal the JSON into the EmployeesInput slice
	if err := json.Unmarshal([]byte(jsonInput), &employees); err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}
	err0 := employeeService.LoadEmployeesFromInput(employees)
	require.NoError(t, err0, "Failed to load employees and schedules from input")
	employeeDB, err1 := employeeService.repo.GetEmployees()
	require.NoError(t, err1, "Failed to load employees list")
	id, err2 := util.GetEmployeeIDByName(employeeDB, "Henny Honore")
	// fmt.Println(id)
	require.NoError(t, err2, "Failed to load employees list")
	monthlySchedule, err3 := employeeService.FetchEmployeeSchedule(id, "March", 2024)
	require.NoError(t, err3, "Failed to fetch the Monthly calendar")
	areEqual, diff := util.CompareMonthlySchedules(schedulesResult, monthlySchedule)
	if !areEqual {
		fmt.Println("Failed to provide the expected Monthly schedule")
		fmt.Println(diff)
	}
}
