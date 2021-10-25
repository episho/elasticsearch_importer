package commands

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"elena/elasticsearch_importer/logic"
	"elena/elasticsearch_importer/model"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const mappings = `{
"settings":{
  "number_of_shards":1,
  "number_of_replicas":0
},
"mappings":{
  "properties": {
    "first_name": {
      "type": "text"
    },
    "last_name": {
      "type": "text"
    },
    "gender": {
      "type": "text"
    },
    "date_of_birth": {
      "type": "date"
    },
    "email": {
      "type": "text"
    },
    "date_of_joining": {
      "type": "date"
    },
    "salary": {
      "type": "long"
    },
    "phone_number": {
      "type": "text"
    }
  }
}
}

`

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
var phoneNumRegex = regexp.MustCompile(`^[1-9]{1}\d{2}[\-]\d{3}[\-][0-9]{4}`)

func newCSVImporterCmd(config *model.Config) *cobra.Command {
	var csvFilePath, csvErrFilePath string
	var numOfWorkers int
	importerCmd := &cobra.Command{
		Use:   "import",
		Short: "Imports all rows from the csv in ES",
		RunE: func(cmd *cobra.Command, args []string) error {
			csvFilePath, err := cmd.Flags().GetString("csvFilePath")
			if err != nil {
				return fmt.Errorf("failed to return the string value of csvFilePath flag: %v", err)
			}

			csvErrFilePath, err := cmd.Flags().GetString("csvErrFilePath")
			if err != nil {
				return fmt.Errorf("failed to return the string value of csvErrFilePath flag: %v", err)
			}

			numOfWorkers, err := cmd.Flags().GetInt("numOfWorkers")
			if err != nil {
				return fmt.Errorf("failed to return the int value of numOfWorkers flag: %v", err)
			}

			csvfile, err := os.Open(csvFilePath)
			if err != nil {
				log.Fatalln("Couldn't open the csv file", err)
			}
			defer csvfile.Close()

			execute(&numOfWorkers, csvfile, csvErrFilePath, config)

			return nil
		},
	}

	importerCmd.PersistentFlags().StringVar(
		&csvFilePath,
		"csvFilePath",
		"csv/employees.csv",
		"csv file path",
	)

	importerCmd.PersistentFlags().StringVar(
		&csvErrFilePath,
		"csvErrFilePath",
		"csv/errors.csv",
		"err csv file path",
	)

	importerCmd.PersistentFlags().IntVar(
		&numOfWorkers,
		"numOfWorkers",
		2,
		"number of workers",
	)

	return importerCmd
}

func execute(
	numWorkers *int,
	filename *os.File,
	csvErrFilePath string,
	config *model.Config,
) {
	ctx := context.Background()

	// create the csv file with errors
	file, err := os.OpenFile(csvErrFilePath, os.O_CREATE|os.O_WRONLY, 0777)
	defer file.Close()
	if err != nil {
		os.Exit(1)
	}

	csvWriter := csv.NewWriter(file)

	// Parse the file
	r := csv.NewReader(filename)
	r.Comma = ';'

	headers, err := r.Read()
	if err != nil {
		logrus.Errorf("read parsers err: %v", err)
	}

	errFileHeader := append(headers, "error")
	err = csvWriter.Write(errFileHeader)
	if err != nil {
		logrus.Error("failed to write headers to the error output file")
		return
	}

	// if index does not exist creates the 'employee' index
	exists, err := config.ESClient.IndexExists("employees").Do(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		_, err := config.ESClient.CreateIndex("employees").BodyString(mappings).Do(ctx)
		if err != nil {
			log.Fatalf("CreateIndex() ERROR: %v", err)
		}
	}

	employeeSvc := logic.NewEmployeeService(config.ESClient)

	var wgWorkers sync.WaitGroup
	var wgCollectors sync.WaitGroup

	jobs := make(chan *model.Job)
	errors := make(chan *model.ErrRow)

	for i := 0; i < *numWorkers; i++ {
		wgWorkers.Add(1)

		go worker(ctx, &wgWorkers, employeeSvc, jobs, errors)
	}

	wgCollectors.Add(1)

	go sumErrors(ctx, &wgCollectors, csvWriter, errors)

	// sending job to the workers
	sendJobs(r, jobs)

	// wait until all jobs are done
	wgWorkers.Wait()
	close(errors)

	logrus.Info("Import finished.")
	wgCollectors.Wait()
}

func sendJobs(r *csv.Reader, jobs chan *model.Job) {
	defer close(jobs)

	rowNum := 0
	for {
		// read each record from csv
		row, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			logrus.WithError(err).Errorln("read row")
			break
		}

		rowNum++
		jobs <- &model.Job{
			Row:    row,
			RowNum: rowNum,
		}
	}
}

func worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	employeeSvc logic.EmployeeService,
	jobs <-chan *model.Job,
	errors chan<- *model.ErrRow,
) {
	defer wg.Done()

	for j := range jobs {
		employee, err := parseRow(j)
		if err != nil {
			logrus.Error(err)
			errors <- &model.ErrRow{
				RowID: j.RowNum,
				Error: err,
				Job:   *j,
			}
			continue
		}

		err = employeeSvc.InsertEmployee(ctx, employee)
		fmt.Println(employee)
		if err != nil {
			logrus.Errorf("failed to insert row num %d err: %v", j.RowNum, err)
			errors <- &model.ErrRow{
				RowID: j.RowNum,
				Error: err,
				Job:   *j,
			}
		}
	}

	logrus.Info("worker done")
}

func sumErrors(
	ctx context.Context,
	wg *sync.WaitGroup,
	csvWriter *csv.Writer,
	errors <-chan *model.ErrRow,
) {
	defer wg.Done()

	for err := range errors {
		var row []string
		row = append(row, err.Job.Row[:7]...)

		errRow := []string{err.Error.Error()}
		row = append(row, errRow...)

		err := csvWriter.Write(row)

		if err != nil {
			logrus.Errorf("failed to write in the csv file %w", err)
		}
	}
	csvWriter.Flush()
}

func parseRow(job *model.Job) (*model.Employee, error) {
	var err error

	id, err := strconv.Atoi(job.Row[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert id %s from string to int", job.Row[0])
	}

	firstName := job.Row[1]
	lastName := job.Row[2]

	gender := strings.ToUpper(job.Row[3])
	if !validateGender(gender) {
		return nil, fmt.Errorf("invalid gender short form %s", gender)
	}

	email := job.Row[4]
	if !validateEmail(email) {
		return nil, fmt.Errorf("invalid email: %s", email)
	}

	dateOfBirth, err := time.Parse("1/2/2006", job.Row[5])
	if err != nil {
		return nil, fmt.Errorf("failed to parse date_of_birth date err: %v", err)
	}

	dateOfJoining, err := time.Parse("1/2/2006", job.Row[6])
	if err != nil {
		return nil, fmt.Errorf("failed to parse date_of_joining date err: %v", err)
	}

	salary, err := strconv.ParseFloat(job.Row[7], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert salary %s from string to float64", job.Row[7])
	}

	phoneNum := job.Row[8]
	if !validatePhoneNumber(phoneNum) {
		return nil, fmt.Errorf("invalid phone number: %s", phoneNum)
	}

	return &model.Employee{
		ID:            id,
		FirstName:     firstName,
		LastName:      lastName,
		Gender:        gender,
		DateOfBirth:   dateOfBirth,
		Email:         email,
		DateOfJoining: dateOfJoining,
		Salary:        salary,
		PhoneNumber:   phoneNum,
	}, nil
}

func validateGender(gender string) bool {
	switch gender {
	case "M":
		return true
	case "F":
		return true
	}

	return false
}

func validateEmail(email string) bool {
	if len(email) < 3 && len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

func validatePhoneNumber(phoneNumber string) bool {
	return phoneNumRegex.MatchString(phoneNumber)
}
