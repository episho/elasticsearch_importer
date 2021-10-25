package main

import (
	"elena/elasticsearch_importer/commands"
	"elena/elasticsearch_importer/model"

	"github.com/olivere/elastic"
)

func main() {
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL("http://0.0.0.0:9200"),
	)
	if err != nil {
		panic(err)
	}

	rootCmd := commands.New(&model.Config{
		ESClient: client,
	})

	rootCmd.Execute()
}
