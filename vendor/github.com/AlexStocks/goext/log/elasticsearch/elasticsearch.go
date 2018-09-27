// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// 2017-04-02 02:04
// package gxelasticsearch provides a Elasticsearch driver
package gxelasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

import (
	"github.com/AlexStocks/goext/log"
	// Log "github.com/AlexStocks/log4go"
	"github.com/juju/errors"
	es "gopkg.in/olivere/elastic.v3"
)

type EsClient struct {
	*es.Client
	bulk *es.BulkService
}

func CreateEsClient(hosts []string) (EsClient, error) {
	var esClient EsClient
	// Create a client
	// snif的作用是根据部分url获取整个集群的urls
	client, err := es.NewClient(es.SetURL(hosts[:]...), es.SetSniff(true), es.SetHealthcheck(true))
	if err == nil {
		esClient.Client = client
	}

	return esClient, err
}

// https://github.com/olivere/elastic/issues/457
func buildEsIndexSettings(shardNum, replicaNum, refreshInterval int32) string {
	return fmt.Sprintf(`{
		"settings" : {
			"number_of_shards": %d,
			"number_of_replicas": %d,
			"refresh_interval": "%ds"
		}
	}`, shardNum, replicaNum, refreshInterval)
}

func buildEsIndexSettingsWithTimestamp(shardNum, replicaNum, refreshInterval int32,
	indexType, timestampField, timestampFormat string) string {

	return fmt.Sprintf(`{
		"settings" : {
			"number_of_shards": %d,
			"number_of_replicas": %d,
			"refresh_interval": "%ds"
		},
		"mappings" : {
	    "%s" : {
				"properties" : {
					"%s" : {
						"type":"date",
						"format":"%s"
					}
				}
	    }
		}
	}`, shardNum, replicaNum, refreshInterval, indexType, timestampField, timestampFormat)
}

func (ec EsClient) CreateEsIndex(index string, shardNum, replicaNum, refreshInterval int32) error {
	var (
		err    error
		exists bool
		body   string
		ctx    context.Context
	)

	ctx = context.Background()
	exists, err = ec.IndexExists(index).DoC(ctx)
	if err != nil {
		return errors.Annotatef(err, "CreateRcIndex(index:%s, shardNum:%s, replicaNum:%d, refreshInterval:%d)",
			index, shardNum, replicaNum, refreshInterval)
	}
	if exists {
		return nil
	}

	body = buildEsIndexSettings(shardNum, replicaNum, refreshInterval)
	_, err = ec.CreateIndex(index).BodyString(body).DoC(ctx)
	if err != nil {
		return errors.Annotatef(err, "CreateEsIndex(body:%s)", body)
	}

	return nil
}

func (ec EsClient) CreateEsIndexWithTimestamp(index string, shardNum, replicaNum, refreshInterval int32,
	indexType, timestampField, timestampFormat string) error {
	var (
		err    error
		exists bool
		body   string
		ctx    context.Context
	)

	ctx = context.Background()
	exists, err = ec.IndexExists(index).DoC(ctx)
	if err != nil {
		return errors.Annotatef(err, "CreateRcIndex(index:%s, shardNum:%s, replicaNum:%d, refreshInterval:%d)",
			index, shardNum, replicaNum, refreshInterval)
	}
	if exists {
		return nil
	}

	body = buildEsIndexSettingsWithTimestamp(shardNum, replicaNum, refreshInterval, indexType, timestampField, timestampFormat)
	_, err = ec.CreateIndex(index).BodyString(body).DoC(ctx)
	if err != nil {
		return errors.Annotatef(err, "CreateEsIndex(body:%s)", body)
	}

	return nil
}

func (ec EsClient) DeleteEsIndex(index string) error {
	var (
		err error
		ctx context.Context
	)

	ctx = context.Background()
	_, err = ec.DeleteIndex(index).DoC(ctx)
	if err != nil {
		return errors.Annotatef(err, "DeleteEsIndex(index:%s)", index)
	}

	return nil
}

// [template](https://github.com/inloco/kafka-elasticsearch-injector/blob/master/src/elasticsearch/elasticsearch_test.go#L39)
//`
//{
//	"template": "my-topic-*",
//	"settings": {},
//	"mappings": {
//	  "my-topic": {
//		"_source": {
//		  "enabled": "true"
//		},
//		"dynamic_templates": [
//		  {
//			"strings": {
//			  "mapping": {
//				"index": "not_analyzed",
//				"type": "string"
//			  },
//			  "match_mapping_type": "string"
//			}
//		  }
//		],
//		"properties": {
//		  "id": {
//		  	"type": "keyword"
//		  }
//		}
//	  }
//	},
//	"aliases": {}
//}
//`
// set mapping & settings
func (ec EsClient) SetTemplate(index, template string, force bool) error {
	var (
		err   error
		exist bool
	)
	if !force {
		exist, err = ec.Client.IndexTemplateExists(index).DoC(context.Background())
		if err != nil {
			return errors.Annotatef(err, "client.IndexTemplateExists(index:%s)", index)
		}
	} else {
		exist = false
	}
	if !exist {
		_, err := ec.IndexPutTemplate(index).BodyString(template).DoC(context.Background())
		if err != nil {
			return errors.Annotatef(err, "IndexPutTemplate(index:%s)", index)
		}
	}

	return nil
}

// InsertWithDocId 插入@msg
// !!! 如果@msg的类型是string 或者 []byte，则被当做Json String类型直接存进去
func (ec EsClient) Insert(index, typ string, msg interface{}) error {
	var (
		err      error
		ok       bool
		msgBytes []byte
		ctx      context.Context
	)

	// https://github.com/olivere/elastic/issues/127
	// Elasticsearch can create an identifier for you, automatically.
	// _, err = ec.Index().Index(index).Type(typ).Id(1).BodyJson(msg).Do()
	ctx = context.Background()
	switch msg.(type) {
	case string:
		_, err = ec.Index().Index(index).Type(typ).BodyString(msg.(string)).DoC(ctx)
		if err != nil {
			return errors.Annotatef(err, "Insert(index:%s, type:%s, msg:%s)", index, typ, msg)
		}

	default:
		if msgBytes, ok = msg.([]byte); ok {
			_, err = ec.Index().Index(index).Type(typ).BodyString(string(msgBytes)).DoC(ctx)
			if err != nil {
				return errors.Annotatef(err, "Insert(index:%s, type:%s, msg:%s)", index, typ, (string)(msgBytes))
			}
		} else {
			_, err = ec.Index().Index(index).Type(typ).BodyJson(msg).DoC(ctx)
			if err != nil {
				return errors.Annotatef(err, "Insert(index:%s, type:%s, msg:%#v)", index, typ, msg)
			}
		}
	}

	return nil
}

// InsertWithDocId 插入@msg时候指定@docID
// !!! 如果@msg的类型是string 或者 []byte，则被当做Json String类型直接存进去
func (ec EsClient) InsertWithDocId(index, typ, docID string, msg interface{}) error {
	var (
		err      error
		ok       bool
		msgBytes []byte
		ctx      context.Context
	)

	ctx = context.Background()
	switch msg.(type) {
	case string:
		_, err = ec.Index().Index(index).Type(typ).Id(docID).BodyString(msg.(string)).DoC(ctx)
		if err != nil {
			return errors.Annotatef(err, "InsertWithDocId(index:%s, type:%s, docID:%s, msg:%s)",
				index, typ, docID, msg)
		}

	default:
		if msgBytes, ok = msg.([]byte); ok {
			_, err = ec.Index().Index(index).Type(typ).Id(docID).BodyString(string(msgBytes)).DoC(ctx)
			if err != nil {
				return errors.Annotatef(err, "InsertWithDocId(index:%s, type:%s, docID:%s, msg:%s)",
					index, typ, docID, (string)(msgBytes))
			}
		} else {
			_, err = ec.Index().Index(index).Type(typ).Id(docID).BodyJson(msg).DoC(ctx)
			if err != nil {
				return errors.Annotatef(err, "InsertWithDocId(index:%s, type:%s, docID:%s, msg:%#v)",
					index, typ, docID, msg)
			}
		}
	}

	return nil
}

// BulkInsert 批量插入@arr
// !!! 如果@arr[0]的类型是string 或者 []byte，则被当做Json String类型直接存进去
// https://www.elastic.co/guide/en/elasticsearch/guide/current/bulk.html
// A good place to start is with batches of 1,000 to 5,000 documents
func (ec EsClient) BulkInsert(index, typ string, arr []interface{}) error {
	var (
		err  error
		ctx  context.Context
		bulk *es.BulkService
		rsp  *es.BulkResponse
	)

	if ec.bulk == nil {
		ec.bulk = ec.Bulk()
	}

	bulk = ec.bulk.Index(index).Type(typ)
	for _, e := range arr {
		switch e.(type) {
		case string:
			data := ([]byte)(e.(string))
			bulk.Add(es.NewBulkIndexRequest().Doc((*json.RawMessage)(&data)))
		default:
			if data, ok := e.([]byte); ok {
				bulk.Add(es.NewBulkIndexRequest().Doc((*json.RawMessage)(&data)))
			} else {
				bulk.Add(es.NewBulkIndexRequest().Doc(e))
				// bulk.Add(es.NewBulkIndexRequest().Index("1").Doc(e))
			}
		}
	}
	if bulk.NumberOfActions() <= 0 {
		return fmt.Errorf("bulk.NumberOfActions() = %d", bulk.NumberOfActions())
	}

	ctx = context.Background()
	rsp, err = bulk.DoC(ctx)
	if err != nil {
		return errors.Annotatef(err, "BulkInsert(@arr len:%d)", len(arr))
	}
	if rsp.Errors {
		// for i, f := range rsp.Failed() {
		// 	Log.Warn("index:%d, fail:%s", i, gxlog.PrettyStruct(f))
		// }
		return fmt.Errorf("BulkInsert(@arr len:%d), failed number:%#v, first fail{reason:%#v, fail detail:%#v}",
			len(arr), len(rsp.Failed()), gxlog.PrettyString(rsp.Failed()[0]))
	}

	return nil
}

// Search
// Waiting for es5.x's future sql feature
