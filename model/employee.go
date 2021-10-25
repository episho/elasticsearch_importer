package model

import "time"

// Employee holds all information for the employee
type Employee struct {
	ID            int       `json:"id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Gender        string    `json:"gender"`
	DateOfBirth   time.Time `json:"date_of_birth"`
	Email         string    `json:"email"`
	DateOfJoining time.Time `json:"date_of_joining"`
	Salary        float64   `json:"salary"`
	PhoneNumber   string    `json:"phone_number"`
}

// Job holds the information about the csv row
type Job struct {
	Row    []string
	RowNum int
}

// ErrRow holds the error information that occurred during the csv import
type ErrRow struct {
	RowID int
	Error error
	Job   Job
}
