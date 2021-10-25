# Elasticsearch importer
Inserting records in elasticsearch from csv using goroutines and execute complex queries in elasticsearch.
In case of errors during the import from the csv file we create an error.csv file with all records that
haven't been inserted in elasticsearch.

### Local
This project use golang and elasticsearch and kibana.
Make sure that you install elasticsearch and kibana (unless you use docker)



### Docker
To run elasticsearch and kibana using docker, execute the following command:

```bash
docker-compose up
```

### Usage:

Compile the project using:
```bash
go build
```

Import the csv using the following command:
```bash
./elasticsearch_importer import --csvErrFilePath csv/errors.csv --csvFilePath csv/employees.csv --numOfWorkers 2
```

For finding the employee with the highest salary execute the following command:
```bash
./elasticsearch_importer query highest_salary
```

For listing all the employees that have anniversaries for a certain month and day execute
the following command:
```bash
./elasticsearch_importer query anniversaries --month 7 --day 22
```

For more info:
```bash
./elasticsearch_importer -h
```