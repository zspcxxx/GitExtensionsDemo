package main

import (
	log "Utility/ZapLog"
	"encoding/json"
	"errors"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var quit chan bool
var c MQTT.Client

var topicMap map[string]int64
var tmptopicMap map[string]int64
var m *sync.Mutex
var DIDcount int64

const (
	FIRSTVERSION  = "1"
	SECONDVERSION = "2"
)

func comparTime() (response HistrDataTime, flag bool) {
	s := time.Now().Minute() * 60 + time.Now().Second()
	if s > 3480 && s < 3600 {
	//if s > 10 && s < 60 {
		t := time.Now().Unix()
		h := time.Now().Hour() + 1
		if h == 24 {
			response.Hour = 0
			addday, _ := time.ParseDuration("24h")
			thisday := time.Unix(t, 0)
			response.Date = thisday.Add(addday).Format("2006-01-02")

		} else {
			response.Hour = h
			response.Date = time.Unix(t, 0).Format("2006-01-02")
		}
		return response, true
	}
	return response, false
}

func topicCheck() {
	tmptopicMap = make(map[string]int64)
	DIDcount = 0
	t := time.NewTimer(5 * time.Minute)
	//t := time.NewTimer(15 * time.Second)   //for debug
	for {
		select {
		case <-t.C:
			err := readLatestAddDevice(&DIDcount)
			if err != nil{
				log.ErrorS("topicCheck", "readLatestAddDevice", err)
			}
			for k := range topicMap{
				for m := range tmptopicMap{
					if strings.Compare(k, m) == 0 {
						tmpDid := tmptopicMap[m]
						addTopicMap(m, tmpDid)
						subscribeTopic0(m)
					}
				}
			}
			for n := range tmptopicMap {
				delete(tmptopicMap, n)
			}
			t.Reset(5 * time.Minute)
			//t.Reset(15 * time.Second)  //for debug
		}
	}
}

var onLost MQTT.ConnectionLostHandler = func(client MQTT.Client, e error) {
	log.ErrorS("MqttClient", "onLost", e)
	checkErr("onLost", e, true)
}

var msgRecv0 MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	defer func() {
		if err := recover(); err != nil {
			log.ErrorS("msgRecv0", "server crash: ", err)
			log.Error("msgRecv0", zap.String("stack: ", string(debug.Stack())))
		}
	}()
	for i := range msg.Payload() {
		payStr := string(msg.Payload()[i])

		if i == 8 && payStr == "2"{
			var devRequest DeviceVerTwoRequest
			var devMarsh DeviceReportDataVerTwo

			err := json.Unmarshal(msg.Payload(), &devMarsh)
			if err != nil{
				log.ErrorS("MQTT.MessageHandler", "json.Unmarshal->devRequest", err)
			}
			t, ok := topicMap[msg.Topic()]
			if ok {
				devRequest.DID = t
				devRequest.NewData = devMarsh
				if err := DeviceDataUpdateVerTwo(devRequest); err != nil{
					log.ErrorS("MQTT.MessageHandler", "DeviceDataUpdateVerTwo", err)
				}
			}

		}
	}

	//todo implement msg handle
	var DevDataJu DataIdentify
	var DevRepData DeviceReportInfoLevel
	var DevNewData AddUpdateRequestData
	//var DeviceAddData AddDTRequestData

	err := json.Unmarshal([]byte(msg.Payload()), &DevDataJu)
	//接收MQTT数据回显
	log.InfoS("MQTT.MessageHandler", "Topic", string(msg.Topic()))
	log.InfoS("MQTT.MessageHandler", "Payload", string(msg.Payload()))
	if err != nil{
		log.ErrorS("MQTT.MessageHandler", "json.Unmarshal->DevDataJu", err)
	}
	if DevDataJu.Tp == 2 {
		err := json.Unmarshal([]byte(msg.Payload()), &DevRepData)
		if err != nil{
			log.ErrorS("MQTT.MessageHandler", "json.Unmarshal->DevRepData", err)
		}
		t, ok := topicMap[msg.Topic()]
		if ok {
			DevNewData.DID = t
			DevNewData.NewData = DevRepData
			//判断是否需要更新历史数据
			//Time , updateFlag := comparTime()
			//if updateFlag {
			//	err := DeviceHistrDataUpdate(DevNewData, Time)
			//	if err != nil{
			//		log.ErrorS("MQTT.MessageHandler", "DeviceHistrDataUpdate", err)
			//	}
			//}
			//更新实时数据
			i := 0
			i++
			err := DeviceDataAddUpdate(DevNewData)  //新旧数据对比
			if err != nil{
				log.ErrorS("MQTT.MessageHandler", "DeviceDataAddUpdate", err)
			}
			//DeviceAddData.Data = string(b.ResponseData)
			//DeviceAddData.DID = t
			//err = DeviceDataInput(DeviceAddData)	//更新后数据写入
			//if err != nil{
			//	log.ErrorS("MQTT.MessageHandler", "DeviceDataInput", err)
			//}
		}
	}
}

func addTopicMap(topic string, id int64)  {
	m.Lock()
	topicMap[topic] = id
	m.Unlock()
}

func removeTopicMap(topic string)  {
	m.Lock()
	delete(topicMap, topic)
	m.Unlock()
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.ErrorS("getLocalIP", "error", err)
		os.Exit(1)
	}
	for _, a := range addrs {
		//判断是否正确获取到IP
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalMulticast() && !ipnet.IP.IsLinkLocalUnicast() {
			log.InfoS("getLocalIP", "ip", ipnet)
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	checkErr("getLocalIP", err, true)
	return ""
}

func initMQTTClient()  {
	topicMap = make(map[string]int64)
	var Alldev DeviceInformations
	var Onedev []DeviceInformation

	clientID := mqttClientID + getLocalIP()

	opts := MQTT.NewClientOptions().AddBroker(mqttURL).SetClientID(clientID)
	opts.SetUsername(mqttName)
	opts.SetPassword(mqttPassword)
	opts.SetKeepAlive(5 * time.Second)
	//opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(onLost)

	c = MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.ErrorS("initMQTTClient", "connect mqtt broker", token.Error())
		checkErr("initMQTTClient", token.Error(), true)
	}
	log.DebugS("initMQTTClient", "mqttClientID********************************", clientID)
	//todo load all device from database
	response, err := GetDeviceInfo()
	responseJson := generateJSON(OK, response)
	if err != nil{
		log.ErrorS("initMQTTClient", "GetDeviceInfo", err)
		checkErr("initMQTTClient", err, true)
	}
	err = json.Unmarshal([]byte(responseJson), &Alldev)
	if err != nil {
		log.ErrorS("initMQTTClient", "Unmarshal error", err)
		checkErr("initMQTTClient", err, true)
	}
	//todo add all topic to topic map
	for i:= 0; i < Alldev.Data.Count; i++ {
		Onedev = append(Onedev, Alldev.Data.ALLProductInfo[i])
		s := fmt.Sprintf("TELLHOW/%s/%s/COV", Onedev[i].ProductType, Onedev[i].MacNum)
		topicMap[s] = Onedev[i].DID
	}
	//addTopicMap

	for k := range topicMap{
		subscribeTopic0(k)
	}

	quit = make(chan bool)
	<-quit
}

func quitMqtt()  {
	for k := range topicMap{
		unsubscribeTopic(k)
	}
	c.Disconnect(250)
}

func unsubscribeTopic(topic string) error {
	if c == nil{
		log.ErrorS("unsubscribeTopic", "unsubscribe topic", "client is nil")
		return errors.New("client is nil")
	}
	if token := c.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		log.ErrorS("unsubscribeTopic", "unsubscribe topic", token.Error())
		return token.Error()
	}
	return nil
}

func subscribeTopic0(topic string) error {
	if c == nil{
		log.ErrorS("subscribeTopic0", "unsubscribe topic", "client is nil")
		return errors.New("client is nil")
	}
	if token := c.Subscribe(topic, 0, msgRecv0); token.Wait() && token.Error() != nil {
		log.ErrorS("subscribeTopic", "Subscribe topic", token.Error())
		return token.Error()//todo need improve when subscribe failed
	}
	return nil
}

func publishTopic0(data PublishData) error {
	if c == nil{
		log.ErrorS("publishTopic0", "unsubscribe topic", "client is nil")
		return errors.New("client is nil")
	}

	log.InfoS("publishTopic0", "topic", data.Topic, "payload", data.Payload)
	if token := c.Publish(data.Topic, 0, false, data.Payload); token.Wait() && token.Error() != nil {
		log.ErrorS("publishTopic0", "Publish topic", token.Error())
		return token.Error()
	}

	return nil
}