package commands

import (
	"elena/elasticsearch_importer/model"

	"github.com/olivere/elastic"
	"github.com/spf13/cobra"
)

func New(conf *model.Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "run",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client, err := elastic.NewClient(
				elastic.SetSniff(false),
				elastic.SetURL("http://0.0.0.0:9200"),
			)
			if err != nil {
				return err
			}

			conf.ESClient = client

			return nil
		},
	}

	importer := newCSVImporterCmd(conf)
	rootCmd.AddCommand(importer)

	queries := newQueriesCmd(conf)
	rootCmd.AddCommand(queries)

	return rootCmd
}
