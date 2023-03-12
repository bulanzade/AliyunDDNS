package main

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/go-co-op/gocron"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	AccessIdKey	string
	AccessSecret string
	SubDomain string
	Domain string
}

func readConfig() (config Config, err error) {
	// 读取配置文件
	readBytes, err := os.ReadFile("config.json")
	if err != nil {
		return
	}
	err = json.Unmarshal(readBytes, &config)
	log.Printf("Config: %s\n", config)
	return
}

func updateDomainRecord(config Config) {
	// 获取本机ip
	res, err := http.Get("https://6.ipw.cn")
	if err != nil {
		log.Fatal("Error when getting ip: ", err)
	}
	body, err := io.ReadAll(res.Body)
	ip := string(body)
	res.Body.Close()
	if err != nil {
		log.Fatal("Error when reading response from '6.ipw.cn': ", err)
	}

	client, err := alidns.NewClientWithAccessKey("cn-hangzhou", config.AccessIdKey, config.AccessSecret)
	if err != nil {
		log.Fatal(err)
		return
	}

	// 获取子域名的记录
	subDomainRecordsRequest := alidns.CreateDescribeSubDomainRecordsRequest()
	subDomainRecordsRequest.Scheme = "https"
	subDomainRecordsRequest.SubDomain = config.SubDomain + "." + config.Domain
	describeSubDomainRecordsResponse, err := client.DescribeSubDomainRecords(subDomainRecordsRequest)
	if err != nil {
		log.Fatal("Error when getting subDomain info: ", err)
	}
	log.Printf("Current record: %#v\n", describeSubDomainRecordsResponse.DomainRecords)

	//如果子域名记录的ip与本机当前ip不同，就修改子域名解析记录
	if describeSubDomainRecordsResponse.DomainRecords.Record[0].Value != ip {
		//修改子域名解析记录
		updateDomainRecordRequest := alidns.CreateUpdateDomainRecordRequest()
		updateDomainRecordRequest.Scheme = "https"
		updateDomainRecordRequest.Value = ip
		updateDomainRecordRequest.Type = "AAAA"
		updateDomainRecordRequest.RR = config.SubDomain
		updateDomainRecordRequest.RecordId = describeSubDomainRecordsResponse.DomainRecords.Record[0].RecordId
		response, err := client.UpdateDomainRecord(updateDomainRecordRequest)
		if err != nil {
			log.Fatal("Error when updating record", err)
		}
		log.Printf("Update RequestId: %s\n", response.RequestId)
	}
}

func print() {
	fmt.Println("5s")
}

func main() {
	config, err := readConfig()
	if err != nil {
		log.Fatal("Error when reading config: ", err)
	}

	s := gocron.NewScheduler(time.FixedZone("UTC+8", 0))
	_, err = s.Every(10).Minutes().Do(updateDomainRecord, config)
	if err != nil {
		log.Fatal(err)
		return 
	}
	s.StartBlocking()
}
