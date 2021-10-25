package logic

import (
	"context"
	"elena/elasticsearch_importer/model"
	"encoding/json"
	"strconv"

	"github.com/olivere/elastic"
)

type EmployeeService interface {
	InsertEmployee(ctx context.Context, employee *model.Employee) error
}

type EmployeeServiceImpl struct {
	esClient *elastic.Client
}

func NewEmployeeService(esClient *elastic.Client) *EmployeeServiceImpl {
	return &EmployeeServiceImpl{
		esClient: esClient,
	}
}

// InsertEmployee inserts the employee in elasticsearch_importer
func (empSvc *EmployeeServiceImpl) InsertEmployee(
	ctx context.Context,
	employee *model.Employee,
) error {
	_, err := empSvc.esClient.Index().Type("_doc").
		Index("employees").
		Id(strconv.Itoa(employee.ID)).
		BodyJson(employee).
		Do(ctx)
	return err
}

// FindEmployeeWithTheHighestSalary finds the employee with the highest salary
func (empSvc *EmployeeServiceImpl) FindEmployeeWithTheHighestSalary(
	ctx context.Context,
) (*model.Employee, error) {
	var employee model.Employee

	res, err := empSvc.esClient.Search().
		Sort("salary", false).Size(1).RestTotalHitsAsInt(true).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	for _, hit := range res.Hits.Hits {
		jsonStr, err := hit.Source.MarshalJSON()
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(jsonStr, &employee)
		if err != nil {
			return nil, err
		}
	}

	return &employee, nil
}

// FindEmployeesAnniversaries finds all employees that have anniversaries
func (empSvc *EmployeeServiceImpl) FindEmployeesAnniversaries(
	ctx context.Context,
	month, day int,
) ([]*model.Employee, error) {
	var employees []*model.Employee
	parms := map[string]interface{}{
		"day":   day,
		"month": month,
	}

	generalQ := elastic.NewBoolQuery()

	generalQ = generalQ.Filter(
		elastic.NewScriptQuery(
			elastic.NewScriptInline(
				"doc['date_of_joining'].value.getMonthValue() == params['month']  && doc['date_of_joining'].value.getDayOfMonth() == params['day']").
				Params(parms),
		),
	)

	res, err := empSvc.esClient.Search().
		Index("employees").
		Query(generalQ).
		RestTotalHitsAsInt(true).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	for _, hit := range res.Hits.Hits {
		jsonStr, err := hit.Source.MarshalJSON()
		if err != nil {
			return nil, err
		}

		var employee model.Employee
		err = json.Unmarshal(jsonStr, &employee)
		if err != nil {
			return nil, err
		}

		employees = append(employees, &employee)
	}

	return employees, err
}
