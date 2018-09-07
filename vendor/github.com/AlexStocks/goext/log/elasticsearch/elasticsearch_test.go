package gxelasticsearch

import (
	"encoding/json"
	"strconv"
	"testing"
)

type EsConf struct {
	ShardNum        int32
	ReplicaNum      int32
	RefreshInterval int32
	EsHosts         []string

	PushIndex string
	PushType  string
}

type Doc struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// go test -v -run Insert
// === RUN   TestEsClient_Insert
// --- PASS: TestEsClient_Insert (1.94s)
// === RUN   TestEsClient_InsertWithString
// --- PASS: TestEsClient_InsertWithString (1.87s)
// PASS
// ok  	github.com/AlexStocks/goext/log/elasticsearch	3.824s

// go test -v -run Insert$
// === RUN   TestEsClient_Insert
// --- PASS: TestEsClient_Insert (1.93s)
// PASS
// ok  	github.com/AlexStocks/goext/log/elasticsearch	1.942s
// go test -v -run Insert$
// 此处如果不加$, 则
func TestEsClient_Insert(t *testing.T) {
	var (
		err    error
		esConf EsConf
		client EsClient
		doc    Doc
		key    string
	)

	esConf = EsConf{
		ShardNum:        5,
		ReplicaNum:      0,
		RefreshInterval: 1,
		EsHosts: []string{
			"http://119.81.218.90:5858",
		},
		PushIndex: "dokidoki-push-test",
		PushType:  "gopush",
	}

	// Create a client
	client, err = CreateEsClient(esConf.EsHosts)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the index again
	err = client.DeleteEsIndex(esConf.PushIndex)
	if err != nil {
		t.Log(err)
	}

	// Create an index
	err = client.CreateEsIndex(esConf.PushIndex, esConf.ShardNum, esConf.ReplicaNum, esConf.RefreshInterval)
	if err != nil {
		t.Fatal(err)
	}

	// Add some documents
	doc = Doc{
		Subject: "Invitation",
		Message: "Would you like to visit me in Berlin in October?",
	}
	err = client.Insert(esConf.PushIndex, esConf.PushType, doc)
	if err != nil {
		t.Fatal(err)
	}

	doc = Doc{
		Subject: "doc-subject1",
		Message: "doc-msg1",
	}
	key = doc.Subject
	err = client.InsertWithDocId(esConf.PushIndex, esConf.PushType, key, doc)
	if err != nil {
		t.Fatal(err)
	}

	// 此处由于key不变，这个doc值会覆盖上面的doc值
	doc = Doc{
		Subject: "doc-subject2",
		Message: "doc-msg2",
	}
	err = client.InsertWithDocId(esConf.PushIndex, esConf.PushType, key, doc)
	if err != nil {
		t.Fatal(err)
	}

}

// go test -v -run InsertWithString
func TestEsClient_InsertWithString(t *testing.T) {
	var (
		err       error
		esConf    EsConf
		client    EsClient
		doc       Doc
		docString []byte
		key       string
	)

	esConf = EsConf{
		ShardNum:        5,
		ReplicaNum:      0,
		RefreshInterval: 1,
		EsHosts: []string{
			"http://119.81.218.90:5858",
		},
		PushIndex: "dokidoki-push-test",
		PushType:  "gopush",
	}

	// Create a client
	client, err = CreateEsClient(esConf.EsHosts)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the index again
	err = client.DeleteEsIndex(esConf.PushIndex)
	if err != nil {
		t.Log(err)
	}

	// Create an index
	err = client.CreateEsIndex(esConf.PushIndex, esConf.ShardNum, esConf.ReplicaNum, esConf.RefreshInterval)
	if err != nil {
		t.Fatal(err)
	}

	// Add some documents
	doc = Doc{
		Subject: "Invitation",
		Message: "Would you like to visit me in Berlin in October?",
	}
	docString, _ = json.Marshal(doc)
	err = client.Insert(esConf.PushIndex, esConf.PushType, docString)
	if err != nil {
		t.Fatal(err)
	}

	doc = Doc{
		Subject: "doc-subject1",
		Message: "doc-msg1",
	}
	key = doc.Subject
	docString, _ = json.Marshal(doc)
	err = client.InsertWithDocId(esConf.PushIndex, esConf.PushType, key, docString)
	if err != nil {
		t.Fatal(err)
	}

	// 此处由于key不变，这个doc值会覆盖上面的doc值
	doc = Doc{
		Subject: "doc-subject2",
		Message: "doc-msg2",
	}
	docString, _ = json.Marshal(doc)
	err = client.InsertWithDocId(esConf.PushIndex, esConf.PushType, key, string(docString))
	if err != nil {
		t.Fatal(err)
	}
}

// go test -v -run BulkInsert
func TestEsClient_BulkInsert(t *testing.T) {
	var (
		err      error
		esConf   EsConf
		client   EsClient
		docArray []interface{}
		doc      Doc
		docBytes []byte
	)

	esConf = EsConf{
		ShardNum:        5,
		ReplicaNum:      0,
		RefreshInterval: 1,
		EsHosts: []string{
			"http://119.81.218.90:5858",
		},
		PushIndex: "dokidoki-push-test",
		PushType:  "gopush",
	}

	// Create a client
	client, err = CreateEsClient(esConf.EsHosts)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the index again
	err = client.DeleteEsIndex(esConf.PushIndex)
	if err != nil {
		t.Log(err)
	}

	// Create an index
	err = client.CreateEsIndex(esConf.PushIndex, esConf.ShardNum, esConf.ReplicaNum, esConf.RefreshInterval)
	if err != nil {
		t.Fatal(err)
	}

	// Add Doc Array
	for i := 0; i < 10; i++ {
		docArray = append(docArray, Doc{
			Subject: "Invitation index:" + strconv.Itoa(i),
			Message: "Would you like to visit me in Berlin in October?",
		})
	}

	err = client.BulkInsert(esConf.PushIndex, esConf.PushType, docArray)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the index again
	err = client.DeleteEsIndex(esConf.PushIndex)
	if err != nil {
		t.Log(err)
	}

	docArray = docArray[:0]
	// Add Doc String Array
	for i := 0; i < 10; i++ {
		doc = Doc{
			Subject: "Invitation index:" + strconv.Itoa(i*10),
			Message: "Would you like to visit me in Berlin in October?",
		}
		docBytes, _ = json.Marshal(doc)
		docArray = append(docArray, docBytes)
	}

	err = client.BulkInsert(esConf.PushIndex, esConf.PushType, docArray)
	if err != nil {
		t.Fatal(err)
	}
}
