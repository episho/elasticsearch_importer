package commands

import (
	"context"
	"fmt"

	"elena/elasticsearch_importer/logic"
	"elena/elasticsearch_importer/model"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newQueriesCmd(config *model.Config) *cobra.Command {
	var month, day int

	queriesCmd := &cobra.Command{
		Use:   "query",
		Short: "Execute elasticsearch_importer queries",
	}

	employeeSvc := logic.NewEmployeeService(config.ESClient)
	ctx := context.Background()

	employeeWithHighestSalaryCmd := &cobra.Command{
		Use:   "highest_salary",
		Short: "Find the employee with the highest salary",
		RunE:  findEmployeeWithHighestSalary(ctx, employeeSvc),
	}

	employeesAnniversariesCmd := &cobra.Command{
		Use:   "anniversaries",
		Short: "Find all employees that have anniversary for certain date",
		RunE:  findEmployeesAnniversaries(ctx, employeeSvc),
	}

	employeesAnniversariesCmd.PersistentFlags().IntVar(
		&month,
		"month",
		5,
		"month",
	)

	employeesAnniversariesCmd.PersistentFlags().IntVar(
		&day,
		"day",
		17,
		"day",
	)

	queriesCmd.AddCommand(employeeWithHighestSalaryCmd)
	queriesCmd.AddCommand(employeesAnniversariesCmd)

	return queriesCmd
}

// find employee with the highest salary
func findEmployeeWithHighestSalary(ctx context.Context, employeeSvc *logic.EmployeeServiceImpl) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// find employees with the highest salary
		employee, err := employeeSvc.FindEmployeeWithTheHighestSalary(ctx)
		if err != nil {
			return fmt.Errorf("failed to find the employee with the highest salary")
		}

		log.Infof(
			"Employee with the highest salary is %s %s",
			employee.FirstName,
			employee.LastName,
		)

		return nil
	}
}

// find employees who have anniversaries on the certain date
func findEmployeesAnniversaries(ctx context.Context, employeeSvc *logic.EmployeeServiceImpl) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		month, err := cmd.Flags().GetInt("month")
		if err != nil {
			return fmt.Errorf("failed to return the int value of month flag: %v", err)
		}

		day, err := cmd.Flags().GetInt("day")
		if err != nil {
			return fmt.Errorf("failed to return the int value of day flag: %v", err)
		}

		employees, err := employeeSvc.FindEmployeesAnniversaries(ctx, month, day)
		if err != nil {
			return fmt.Errorf("failed to find the employees anniversaries err :%v", err)
		}

		if len(employees) == 0 {
			log.Info("No employees anniversaries")
		} else {
			msg := fmt.Sprintf(
				"On %s we have the following employees anniversaries: \n",
				fmt.Sprintf("%d.%d", day, month),
			)
			for _, employee := range employees {
				msg += fmt.Sprintf("- %s %s\n", employee.FirstName, employee.LastName)
			}
			log.Info(msg)
		}

		return nil
	}
}
