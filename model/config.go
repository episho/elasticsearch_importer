package model

import "github.com/olivere/elastic"

type Config struct {
	ESClient *elastic.Client
}
