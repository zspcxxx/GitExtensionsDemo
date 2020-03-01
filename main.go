package main

import (
	g "Utility/GlobalConfig"
	sredis "Utility/RedisHelper"
	"Utility/RegDis"
	log "Utility/ZapLog"
	pb "Utility/pb/MQTTClient" //本服务的pb文件路径
	"flag"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
)

var serviceName string = "MQTTClient"
var logLevel, logName, yamlPath, mqttClientID, mqttName, mqttPassword, covTopic, mqttURL string
var rd = RegDis.GetInstance()
var listen net.Listener
var port int
var RedisCon, Error =sredis.NewRedisConn()

func init() {
	//Linux下默认Service yaml文件路径为：/etc/tellhow/modules/MQTTClient/MQTTClient.yaml
	var projectConf string
	if runtime.GOOS == "windows" {
		projectConf = "c:\\etc\\tellhow\\modules\\" + serviceName + "\\" + serviceName + ".yaml"
	} else {
		projectConf = "/etc/tellhow/modules/" + serviceName + "/" + serviceName + ".yaml"
	}
	flag.StringVar(&yamlPath, "c", projectConf, "config file path,example /etc/tellhow/MQTTClient.yaml")
	flag.StringVar(&logLevel, "l", "debug", "log level: debug, info, error")
	flag.StringVar(&logName, "n", serviceName, "log name: service name")
}

func waitToExit() {
	c := make(chan os.Signal)
	signal.Notify(c)
	for {
		s := <-c
		log.Error("waitToExit", zap.String("get signal", s.String()))
		if s == syscall.SIGINT || s == syscall.SIGKILL || s == syscall.SIGTERM {
			rd.UnRegistService()
			listen.Close()
			quitMqtt()
			quit <- true
			os.Exit(1)
		}
	}

}

func checkErr(errMessage string, err error, isQuit bool) {
	if err != nil {
		log.Error(errMessage, zap.String("get signal",err.Error()))
		if isQuit {
			listen.Close()
			os.Exit(1)
		}
	}
}

func loadInitConfig(yamlPath string) (map[string]interface{}, error) {
	//Linux下默认log路径为：/var/log/tellhow/MQTTClient.log
	var logPath string
	if runtime.GOOS == "windows" {
		logPath = "c:\\log\\tellhow\\" + serviceName + "\\"
	} else {
		logPath = "/var/log/tellhow/" + serviceName + "/"
	}

	err := os.MkdirAll(logPath, 0777)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	logPath = logPath + logName + ".log"

	switch logLevel {
	case "debug":
		log.Init(logPath, zap.DebugLevel)
	case "info":
		log.Init(logPath, zap.InfoLevel)
	case "error":
		log.Init(logPath, zap.ErrorLevel)
	default:
		log.Init(logPath, zap.DebugLevel)
	}
	cfg, err := g.GetNodeMapData("mqtt")
	if err != nil{
		checkErr("Can't get MQTT client config", err, true)
	}

	m := make(map[string]interface{})

	data, err := ioutil.ReadFile(yamlPath)
	checkErr("ReadFile error", err, false)

	err = yaml.Unmarshal([]byte(data), &m)
	checkErr("yamlConfig Unmarshal error", err, true)

	port = m["port"].(int)//读取yaml配置

	mqttClientID = m["mqttClientID"].(string)
	mqttName = cfg["mqttName"].(string)
	mqttPassword = cfg["mqttPassword"].(string)
	covTopic = cfg["covTopic"].(string)
	mqttURL = cfg["mqttURL"].(string)

	return m, err
}

func main()  {
	defer func() {
		if err := recover(); err != nil {
			log.ErrorS("main", "server crash: ", err)
			log.Error("main", zap.String("stack: ", string(debug.Stack())))
		}
	}()

	m = new(sync.Mutex)

	flag.Parse()
	_, err := loadInitConfig(yamlPath)
	if err != nil {
		checkErr("loadInitConfig error", err,true)
	}

	rd.RegistService(serviceName, port)

	go waitToExit()
	go initMQTTClient()
	go topicCheck()

	log.InfoS("SAA server start...","port", port)
	listen, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	checkErr("tcp port error", err, true)
	s := grpc.NewServer()
	pb.RegisterMQTTClientServer(s, newServer())//newServer需要实现proto文件所定义服务接口

	reflection.Register(s)
	err = s.Serve(listen)
	checkErr("ServiceStartFailed error", err, true)
}
