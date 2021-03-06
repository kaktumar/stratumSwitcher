package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
)

// Zookeeper连接超时时间
const zookeeperConnTimeout = 5

// ConfigData 配置数据
type ConfigData struct {
	// UserListAPI 币种对应的用户列表，形如{"btc":"url", "bcc":"url"}
	UserListAPI map[string]string
	// IntervalSeconds 每次拉取的间隔时间
	IntervalSeconds uint

	// Zookeeper集群的IP:端口列表
	ZKBroker []string
	// ZKSwitcherWatchDir Switcher监控的Zookeeper路径，以斜杠结尾
	ZKSwitcherWatchDir string
}

// zookeeperConn Zookeeper连接对象
var zookeeperConn *zk.Conn

// 配置数据
var configData *ConfigData

// 用于等待goroutine结束
var waitGroup sync.WaitGroup

func main() {
	// 解析命令行参数
	configFilePath := flag.String("config", "./config.json", "Path of config file")
	flag.Parse()

	// 读取配置文件
	configJSON, err := ioutil.ReadFile(*configFilePath)

	if err != nil {
		glog.Fatal("read config failed: ", err)
		return
	}

	configData = new(ConfigData)
	err = json.Unmarshal(configJSON, configData)

	if err != nil {
		glog.Fatal("parse config failed: ", err)
		return
	}

	// 若zookeeper路径不以“/”结尾，则添加
	if configData.ZKSwitcherWatchDir[len(configData.ZKSwitcherWatchDir)-1] != '/' {
		configData.ZKSwitcherWatchDir += "/"
	}

	// 建立到Zookeeper集群的连接
	conn, _, err := zk.Connect(configData.ZKBroker, time.Duration(zookeeperConnTimeout)*time.Second)

	if err != nil {
		glog.Fatal("Connect Zookeeper Failed: ", err)
		return
	}

	zookeeperConn = conn

	// 检查并创建StratumSwitcher使用的Zookeeper路径
	err = createZookeeperPath(configData.ZKSwitcherWatchDir)

	if err != nil {
		glog.Fatal("Create Zookeeper Path Failed: ", err)
		return
	}

	// 开始执行任务
	for coin, url := range configData.UserListAPI {
		waitGroup.Add(1)
		go InitUserCoin(coin, url)
	}

	waitGroup.Wait()

	glog.Info("Init User Coin Finished.")
}
