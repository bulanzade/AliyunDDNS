package main

import (
	"encoding/json"
	"errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/go-co-op/gocron"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	AccessIdKey  string
	AccessSecret string
	SubDomain    string
	Domain       string
}

func readConfig(configFile string) (config Config, err error) {
	readBytes, err := os.ReadFile(configFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(readBytes, &config)
	return
}

func getIP() (ip string, err error) {
	res, err := http.Get("https://6.ipw.cn")
	if err != nil {
		return
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	ip = string(body)
	err = res.Body.Close()
	return
}

func getSubDomainRecord(client *alidns.Client, subDomain string, domain string) (record alidns.Record, err error) {
	request := alidns.CreateDescribeSubDomainRecordsRequest()
	request.Scheme = "https"
	request.SubDomain = subDomain + "." + domain
	response, err := client.DescribeSubDomainRecords(request)
	recordSize := len(response.DomainRecords.Record)
	if err == nil {
		if recordSize > 0 {
			record = response.DomainRecords.Record[0]
		} else {
			err = errors.New("record length is 0")
		}
	}
	return
}

func updateSubDomainRecord(client *alidns.Client, recordId string, value string, rr string) (requestId string, err error) {
	request := alidns.CreateUpdateDomainRecordRequest()
	request.Scheme = "https"
	request.Type = "AAAA"
	request.RR = rr
	request.RecordId = recordId
	request.Value = value
	response, err := client.UpdateDomainRecord(request)
	requestId = response.RequestId
	return
}

func ddns() {
	// read config
	config, err := readConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to read config file: %#v\n", err)
	}
	log.Printf("Current config: %#v\n", config)
	// create client
	client, err := alidns.NewClientWithAccessKey("cn-hangzhou", config.AccessIdKey, config.AccessSecret)
	if err != nil {
		log.Fatalf("Failed to create client: %#v\n", err)
	}
	// get ip
	ip, err := getIP()
	if err != nil {
		log.Printf("Failed to get ip: %#v\n", err)
		return
	}
	log.Printf("Current ip: %s\n", ip)
	// get subdomain record
	record, err := getSubDomainRecord(client, config.SubDomain, config.Domain)
	if err != nil {
		log.Printf("Failed to get subdomain record: %#v\n", err)
		return
	}
	log.Printf("Current record: %#v\n", record)
	// update record if ip has changed
	if record.Value != ip {
		requestId, err := updateSubDomainRecord(client, record.RecordId, ip, record.RR)
		if err != nil {
			log.Printf("Failed to update record: %#v\n", err)
		}
		log.Printf("Update reqeust id: %s\n", requestId)
	}
}

func main() {
	//set log
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	logFile, err := os.OpenFile("ddns.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %#v\n", err)
	}
	log.SetOutput(logFile)
	defer logFile.Close()

	s := gocron.NewScheduler(time.FixedZone("UTC+8", 0))
	_, err = s.Every(5).Minutes().Do(ddns)
	if err != nil {
		log.Fatalln(err)
	}
	s.StartBlocking()
}
