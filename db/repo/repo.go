package db

import (
	"fmt"
	"github.com/lichensio/api_server/db/model"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository interface {
	LoadEmployees(employees []*model.Employee) error
	UpdateEmployee(employee model.Employee) error
	UpdateSchedule(schedule model.Schedule) error
	GetSchedule(employeeID uint, weekType string) ([]model.Schedule, error)
	GetEmployees() ([]model.Employee, error)
	GetEmployeeWithSchedulesByWeekType(employeeID uint, weekType string) (*model.Employee, error)
	CleanupDatabase()
	GetEmployeeByID(id uint, emp *model.Employee) error
	GetEmployeeWithSchedules(id uint) (*model.Employee, error)
	DBCreate() error
	DBDelete() error
	// Define more methods for analytics or other operations as needed
}

type repository struct {
	db *gorm.DB
}

func (r *repository) GetEmployeeByID(id uint, emp *model.Employee) error {
	result := r.db.First(emp, id)
	return result.Error
}

func NewRepositoryWithDB(db *gorm.DB) Repository {
	return &repository{db: db}
}

func NewRepository(dsn string) (Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&model.Employee{}, &model.Schedule{})
	if err != nil {
		return nil, err
	}

	return &repository{db: db}, nil
}

func (r *repository) LoadEmployees(employees []*model.Employee) error {
	return r.db.Create(&employees).Error
}

func (r *repository) UpdateEmployee(employee model.Employee) error {
	return r.db.Save(&employee).Error
}

func (r *repository) UpdateSchedule(schedule model.Schedule) error {
	return r.db.Save(&schedule).Error
}

func (r *repository) GetSchedule(employeeID uint, weekType string) ([]model.Schedule, error) {
	var schedules []model.Schedule
	err := r.db.Where("employee_id = ? AND week_type = ?", employeeID, weekType).Find(&schedules).Error
	return schedules, err
}

func (r *repository) GetEmployees() ([]model.Employee, error) {
	var employees []model.Employee
	err := r.db.Find(&employees).Error
	return employees, err
}

func (r *repository) GetEmployeeWithSchedules(employeeID uint) (*model.Employee, error) {
	var employee model.Employee
	if err := r.db.Preload("Schedules").First(&employee, employeeID).Error; err != nil {
		return nil, err
	}
	return &employee, nil
}

func (r *repository) DBCreate() error {
	if err := r.db.AutoMigrate(&model.Employee{}, &model.Schedule{}); err != nil {
		log.Printf("Failed to migrate database schema: %v", err)
		return err
	}
	log.Println("Database schema migrated successfully.")
	return nil
}

// Additional methods for analytics or other operations can be defined here

// cleanupDatabase deletes all entries from the schedules and then the employees tables.
func (r *repository) CleanupDatabase() {
	// First, delete all entries from the schedules table.
	if err := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Schedule{}).Error; err != nil {
		log.Fatalf("Failed to clean up schedules table: %v", err)
	}

	// Then, delete all entries from the employees table.
	if err := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Employee{}).Error; err != nil {
		log.Fatalf("Failed to clean up employees table: %v", err)
	}
}

func (r *repository) GetEmployeeWithSchedulesByWeekType(employeeID uint, weekType string) (*model.Employee, error) {
	var employee model.Employee

	// Validate weekType input to ensure it's either "A" or "B".
	if weekType != "A" && weekType != "B" {
		return nil, fmt.Errorf("weekType must be either 'A' or 'B', got: %s", weekType)
	}

	// Preload schedules with a condition on the week type.
	if err := r.db.Preload("Schedules", "week_type = ?", weekType).First(&employee, employeeID).Error; err != nil {
		return nil, err
	}

	return &employee, nil
}

func (r *repository) DBDelete() error {
	// Drop `schedules` table first due to the foreign key constraint with `employees`
	if err := r.db.Migrator().DropTable(&model.Schedule{}); err != nil {
		return err
	}
	// Then drop `employees` table
	if err := r.db.Migrator().DropTable(&model.Employee{}); err != nil {
		return err
	}
	return nil
}
