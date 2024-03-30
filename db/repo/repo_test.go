package db

import (
	"fmt"
	"github.com/lichensio/api_server/db/model"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestDB initializes the test database, returns a gorm.DB instance and a cleanup function.
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	err := godotenv.Load() // Adjust to the correct path to your .env file
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	cleanup := func() {
		db.Migrator().DropTable(&model.Schedule{}, &model.Employee{})
	}

	// Prepare the database: clean existing data and migrate
	cleanup()
	err = db.AutoMigrate(&model.Employee{}, &model.Schedule{})
	require.NoError(t, err)

	return db, cleanup
}

// Assuming repository and other necessary structures are correctly defined above.

func TestLoadEmployees(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := &repository{db: db} // Adjust according to how you instantiate the repository
	currentTime := time.Now().UTC()

	employees := []*model.Employee{
		{Name: "John Doex", StartDate: currentTime},
		{Name: "Jane Doe", StartDate: currentTime},
	}

	err := repo.LoadEmployees(employees)
	require.NoError(t, err)

	var dbEmployees []model.Employee
	err = db.Find(&dbEmployees).Error
	require.NoError(t, err)

	assert.Len(t, dbEmployees, len(employees), "The number of loaded employees should match")
	for _, emp := range dbEmployees {
		assert.True(t, emp.StartDate.Equal(currentTime) || emp.StartDate.Before(currentTime.Add(time.Minute)), "The StartDate should closely match the current time")
	}
}

func TestGetEmployees(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := &repository{db: db} // Adjust according to how you instantiate the repository

	repo.CleanupDatabase() // Assuming this properly cleans the test database
	currentTime := time.Now().UTC()

	expectedEmployees := []model.Employee{
		{Name: "John Doex", StartDate: currentTime},
		{Name: "Jane Doe", StartDate: currentTime},
	}

	for _, emp := range expectedEmployees {
		require.NoError(t, db.Create(&emp).Error)
	}

	employees, err := repo.GetEmployees()
	require.NoError(t, err)
	assert.Len(t, employees, len(expectedEmployees))
	for _, emp := range employees {
		empDate := emp.StartDate.Truncate(24 * time.Hour) // Truncate time part
		curDate := currentTime.Truncate(24 * time.Hour)   // Truncate time part
		assert.Equal(t, curDate, empDate, "The StartDate should match the current date")
	}
}

func TestGetEmployeeByID(t *testing.T) {
	db, cleanup := setupTestDB(t) // Set up your test database
	defer cleanup()

	repo := &repository{db: db} // Adjust according to how you instantiate the repository

	// Setup: Create a test employee
	emp := &model.Employee{Name: "Test Employee", StartDate: time.Now()}
	err := repo.LoadEmployees([]*model.Employee{emp})
	require.NoError(t, err)
	require.NotZero(t, emp.ID)

	// Fetch the employee by ID
	var fetchedEmp model.Employee
	err = repo.GetEmployeeByID(emp.ID, &fetchedEmp)
	require.NoError(t, err)
	assert.Equal(t, emp.Name, fetchedEmp.Name)
}

func TestUpdateEmployee(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a repository instance. Use NewRepository if you have a factory function that returns a Repository interface
	repo := &repository{db: db} // Adjust according to how you instantiate the repository

	// Assuming a cleanup method on the repository interface; if not, adapt accordingly
	repo.CleanupDatabase()

	// Setup: Create an employee for testing
	startDate := time.Now().UTC()
	employee := &model.Employee{Name: "John Doe", StartDate: startDate}

	err := repo.LoadEmployees([]*model.Employee{employee})
	require.NoError(t, err, "Failed to load employee")
	require.NotZero(t, employee.ID, "Employee should have an ID after being loaded.")

	// Update the employee's name
	employee.Name = "John Updated"
	err = repo.UpdateEmployee(*employee)
	require.NoError(t, err, "Failed to update employee")

	// Retrieve and verify the updated employee
	var updatedEmployee model.Employee
	err = repo.GetEmployeeByID(employee.ID, &updatedEmployee) // Assuming GetEmployeeByID is correctly implemented
	require.NoError(t, err, "Failed to retrieve updated employee")
	assert.Equal(t, "John Updated", updatedEmployee.Name, "Employee name should be updated")
}

func TestGetSchedule(t *testing.T) {
	db, cleanup := setupTestDB(t) // Assumes setupTestDB is correctly implemented
	defer cleanup()

	// Adjusted to use the NewRepository factory function or similar if available
	repo := &repository{db: db} // Adjust according to how you instantiate the repository

	// Assuming a cleanup method on the repository interface; if not, adapt accordingly
	repo.CleanupDatabase()

	// Setup: Create an employee for testing
	startDate := time.Now().UTC()
	employee := &model.Employee{Name: "Jane Schedule", StartDate: startDate}
	err := repo.LoadEmployees([]*model.Employee{employee})
	require.NoError(t, err)
	require.NotZero(t, employee.ID, "Employee should have an ID after being loaded.")

	// Create schedules for the loaded employee
	formattedStartTime := time.Now().Round(time.Second)
	formattedEndTime := formattedStartTime.Add(8 * time.Hour)
	schedule := model.Schedule{
		EmployeeID: employee.ID,
		WeekType:   "B",
		DayName:    "Tuesday",
		StartTime:  model.CustomTime{Time: formattedStartTime},
		EndTime:    model.CustomTime{Time: formattedEndTime},
	}

	err = repo.UpdateSchedule(schedule)
	require.NoError(t, err)

	// Test: Retrieve the schedule
	schedules, err := repo.GetSchedule(employee.ID, "B")
	require.NoError(t, err)
	require.Len(t, schedules, 1, "Should retrieve exactly one schedule.")

	expectedStartTime := formattedStartTime.Format("15:04:05")
	actualStartTime := schedules[0].StartTime.Time.Format("15:04:05")
	assert.Equal(t, expectedStartTime, actualStartTime, "StartTime should match")

	expectedEndTime := formattedEndTime.Format("15:04:05")
	actualEndTime := schedules[0].EndTime.Time.Format("15:04:05")
	assert.Equal(t, expectedEndTime, actualEndTime, "EndTime should match")
}

func TestUpdateSchedule(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := &repository{db: db} // Adjust according to how you instantiate the repository
	repo.CleanupDatabase()      // Assuming this properly cleans the test database
	// Assuming an employee is already created for this test
	employee := model.Employee{Name: "Test Employee", StartDate: time.Now()}
	if err := db.Create(&employee).Error; err != nil {
		t.Fatalf("Failed to create test employee: %v", err)
	}

	// Create a new schedule to update
	schedule := model.Schedule{
		EmployeeID: employee.ID,
		WeekType:   "A",
		DayName:    "Monday",
		StartTime:  model.CustomTime{Time: time.Now()},
		EndTime:    model.CustomTime{Time: time.Now().Add(8 * time.Hour)},
	}
	if err := db.Create(&schedule).Error; err != nil {
		t.Fatalf("Failed to create test schedule: %v", err)
	}

	// Update the schedule
	schedule.DayName = "Tuesday" // Changing the day to Tuesday
	if err := repo.UpdateSchedule(schedule); err != nil {
		t.Fatalf("Failed to update schedule: %v", err)
	}

	// Retrieve and assert the schedule was updated
	var updatedSchedule model.Schedule
	if err := db.First(&updatedSchedule, schedule.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated schedule: %v", err)
	}

	assert.Equal(t, "Tuesday", updatedSchedule.DayName)
}

func TestGetEmployeeWithSchedules(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := &repository{db: db} // Adjust according to how you instantiate the repository
	repo.CleanupDatabase()      // Assuming this properly cleans the test database
	// Create an employee and their schedule for testing
	employee := model.Employee{Name: "Schedule Employee", StartDate: time.Now()}
	if err := db.Create(&employee).Error; err != nil {
		t.Fatalf("Failed to create test employee: %v", err)
	}

	schedule := model.Schedule{
		EmployeeID: employee.ID,
		WeekType:   "A",
		DayName:    "Monday",
		StartTime:  model.CustomTime{Time: time.Now()},
		EndTime:    model.CustomTime{Time: time.Now().Add(8 * time.Hour)},
	}
	if err := db.Create(&schedule).Error; err != nil {
		t.Fatalf("Failed to create test schedule: %v", err)
	}

	// Retrieve the employee with schedules
	resultEmployee, err := repo.GetEmployeeWithSchedules(employee.ID)
	require.NoError(t, err)

	assert.Equal(t, employee.Name, resultEmployee.Name)
	assert.Len(t, resultEmployee.Schedules, 1)
	assert.Equal(t, "Monday", resultEmployee.Schedules[0].DayName)
}

func TestGetEmployeeWithSchedulesByWeekType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := &repository{db: db} // Adjust according to how you instantiate the repository
	repo.CleanupDatabase()      // Assuming this properly cleans the test database
	// Create and insert a test employee
	currentTime := time.Now().UTC()
	employee := model.Employee{Name: "Employee With Schedules", StartDate: currentTime}
	require.NoError(t, db.Create(&employee).Error)

	// Create and insert schedules for the employee
	aSchedule := model.Schedule{
		EmployeeID: employee.ID,
		WeekType:   "A",
		DayName:    "Monday",
		StartTime:  model.CustomTime{Time: time.Now()},
		EndTime:    model.CustomTime{Time: time.Now().Add(8 * time.Hour)},
	}
	bSchedule := model.Schedule{
		EmployeeID: employee.ID,
		WeekType:   "B",
		DayName:    "Tuesday",
		StartTime:  model.CustomTime{Time: time.Now()},
		EndTime:    model.CustomTime{Time: time.Now().Add(8 * time.Hour)},
	}
	require.NoError(t, db.Create(&aSchedule).Error)
	require.NoError(t, db.Create(&bSchedule).Error)

	// Test fetching the employee with schedules for week type "A"
	empWithSchedulesA, err := repo.GetEmployeeWithSchedulesByWeekType(employee.ID, "A")
	require.NoError(t, err, "Fetching employee with schedules for week type A should not error")
	assert.Len(t, empWithSchedulesA.Schedules, 1, "Employee should have exactly one schedule for week type A")
	assert.Equal(t, "A", empWithSchedulesA.Schedules[0].WeekType, "Schedule week type should be A")

	// Test fetching the employee with schedules for week type "B"
	empWithSchedulesB, err := repo.GetEmployeeWithSchedulesByWeekType(employee.ID, "B")
	require.NoError(t, err, "Fetching employee with schedules for week type B should not error")
	assert.Len(t, empWithSchedulesB.Schedules, 1, "Employee should have exactly one schedule for week type B")
	assert.Equal(t, "B", empWithSchedulesB.Schedules[0].WeekType, "Schedule week type should be B")
}

func TestLoadEmployeeWithMorningAndAfternoonSchedules(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := &repository{db: db} // Adjust according to how you instantiate the repository
	repo.CleanupDatabase()      // Assuming this properly cleans the test database
	// Create and insert a new employee. Note the use of & to get a pointer
	employee := &model.Employee{Name: "Full Week Employee", StartDate: time.Now().UTC()}
	err := repo.LoadEmployees([]*model.Employee{employee})
	require.NoError(t, err, "Failed to load new employee")
	require.NotZero(t, employee.ID, "Employee should have an ID after being loaded.")

	// Define time slots for morning and afternoon schedules
	timeSlots := []struct {
		StartTime time.Time
		EndTime   time.Time
	}{
		{StartTime: time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 0, 0, 12, 0, 0, 0, time.UTC)},  // Morning
		{StartTime: time.Date(0, 0, 0, 13, 0, 0, 0, time.UTC), EndTime: time.Date(0, 0, 0, 17, 0, 0, 0, time.UTC)}, // Afternoon
	}

	// Create schedules for a complete week for both Week A and Week B, for morning and afternoon
	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for _, weekType := range []string{"A", "B"} {
		for _, day := range daysOfWeek {
			for _, slot := range timeSlots {
				schedule := model.Schedule{
					EmployeeID: employee.ID,
					WeekType:   weekType,
					DayName:    day,
					StartTime:  model.CustomTime{Time: slot.StartTime},
					EndTime:    model.CustomTime{Time: slot.EndTime},
				}
				err := repo.UpdateSchedule(schedule)
				require.NoError(t, err, fmt.Sprintf("Failed to load schedule for %s of week %s", day, weekType))
			}
		}
	}

	// Verify that the employee has 28 schedules in total (14 for Week A and 14 for Week B)
	loadedEmployeeWithSchedulesA, err := repo.GetEmployeeWithSchedulesByWeekType(employee.ID, "A")
	require.NoError(t, err, "Failed to retrieve employee with schedules for Week A")
	assert.Len(t, loadedEmployeeWithSchedulesA.Schedules, 14, "Employee should have 14 schedules for Week A")

	loadedEmployeeWithSchedulesB, err := repo.GetEmployeeWithSchedulesByWeekType(employee.ID, "B")
	require.NoError(t, err, "Failed to retrieve employee with schedules for Week B")
	assert.Len(t, loadedEmployeeWithSchedulesB.Schedules, 14, "Employee should have 14 schedules for Week B")
}

// Additional test functions adapted for PostgreSQL
