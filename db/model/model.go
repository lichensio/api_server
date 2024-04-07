package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// CustomTime wraps time.Time for handling PostgreSQL 'time without time zone' fields.
type CustomTime struct {
	time.Time
}

// Scan implements the sql.Scanner interface for CustomTime,
// allowing custom parsing of time data from the database.
func (ct *CustomTime) Scan(value interface{}) error {
	var err error
	switch v := value.(type) {
	case []byte:
		ct.Time, err = time.Parse("15:04:05", string(v))
	case string:
		ct.Time, err = time.Parse("15:04:05", v)
	case time.Time:
		ct.Time = v
	default:
		return fmt.Errorf("cannot scan type %T into CustomTime", value)
	}
	return err
}

// Value implements the driver.Valuer interface for CustomTime,
// allowing custom formatting of time data to the database.
func (ct CustomTime) Value() (driver.Value, error) {
	return ct.Format("15:04:05"), nil
}

// Employee represents an employee record in the database and the JSON structure.
type Employee struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	StartDate time.Time `gorm:"type:date;not null" json:"startDate"`
	// GORM automatically interprets the Schedules slice as a one-to-many relationship based on the foreign key.
	Schedules []Schedule `gorm:"foreignKey:EmployeeID" json:"schedules,omitempty"`
}

// Schedule represents the schedule of an employee, aligning with the schedules table.
type Schedule struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	EmployeeID uint       `gorm:"not null" json:"employeeId"`
	WeekType   string     `gorm:"type:char(1);not null" json:"weekType"`
	DayName    string     `gorm:"type:varchar(10);not null" json:"dayName"`
	StartTime  CustomTime `gorm:"type:time without time zone;not null"` // Custom handling
	EndTime    CustomTime `gorm:"type:time without time zone;not null"` // Custom handling
}

// JSON model

type ScheduleInput struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type WeeklyScheduleInput struct {
	Monday    []ScheduleInput `json:"Monday"`
	Tuesday   []ScheduleInput `json:"Tuesday"`
	Wednesday []ScheduleInput `json:"Wednesday"`
	Thursday  []ScheduleInput `json:"Thursday"`
	Friday    []ScheduleInput `json:"Friday"`
	Saturday  []ScheduleInput `json:"Saturday"`
	Sunday    []ScheduleInput `json:"Sunday"`
}

type EmployeeInput struct {
	Name      string                         `json:"name"`
	StartDate string                         `json:"startDate"`
	Weeks     map[string]WeeklyScheduleInput `json:"weeks"`
}

type EmployeesInput []EmployeeInput

// MonthltSchedule wraps a list of ScheduleEntry items for a single employee.
type MonthlySchedule struct {
	Date        string     `json:"date"`
	DayName     string     `json:"dayName"`
	HolidayName string     `json:"holiday_name"`
	TimeSlots   []TimeSlot `json:"timeSlots"`
}

// TimeSlot represents a single working period within a day.
type TimeSlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Holiday represents a holiday record in the french_holidays table
type Holiday struct {
	HolidayDate time.Time `gorm:"primary_key" json:"holiday_date"`
	HolidayName string    `json:"holiday_name"`
}

type EmployeeHoliday struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EmployeeID  uint      `gorm:"not null;index" json:"employeeId"`
	HolidayDate time.Time `gorm:"type:date;not null" json:"holidayDate"`
	Description string    `gorm:"type:varchar(255)" json:"description"`     // Optional description of the holiday
	WithoutPay  bool      `gorm:"not null;default:false" json:"withoutPay"` // Indicates if the holiday is without pay
}
