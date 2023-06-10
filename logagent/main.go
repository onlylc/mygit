package main

import (
	"fmt"
	"logagent/conf"
	"logagent/etcd"
	"logagent/kafka"
	"logagent/taillog"
	"sync"
	"time"

	"gopkg.in/ini.v1"
)

var (
	cfg = new(conf.AppConf)
)

// logAgent 入口程序

// func run() {
// 	// 读取日志
// 	for {
// 		select {
// 		case line := <-taillog.ReadLog():
// 			kafka.SendToKafka(cfg.KafkaConf.Topic, line.Text)
// 		default:
// 			time.Sleep(time.Second)
// 		}
// 	}

// }

func main() {
	//0. 加载配置文件

	err := ini.MapTo(cfg, "./conf/config.ini")
	if err != nil {
		fmt.Printf("load ini failed, err: %v\n", err)
		return
	}

	// 1. 初始化kafka链接
	err = kafka.Init([]string{cfg.KafkaConf.Address}, cfg.KafkaConf.ChanMaxSize)
	if err != nil {
		fmt.Printf("Init kafka failed, err: %v\n", err)
		return
	}
	fmt.Println("Init kafka success")

	// 初始化etcd

	err = etcd.Init(cfg.EtcdConf.Address, time.Duration(cfg.EtcdConf.TimeTou)*time.Second)
	if err != nil {
		fmt.Printf("Init etcd failed, err: %v\n", err)
		return
	}
	fmt.Println("Init etcd success")

	// 为了实现每个logagent都拉取自己独有的配置 需要自己的ip地址作为区分
	// 2. 从etcd中获取日志收集项的配置信息
	// ipStr, err := utils.GetboundIP()
	// if err != nil {
	// 	panic(err)

	// }
	apStr := "127.0.0.1"
	etcdConfKey := fmt.Sprintf(cfg.EtcdConf.Key, apStr)
	logEntryConf, err := etcd.GetConf(etcdConfKey)

	if err != nil {
		fmt.Printf("etcd.GetConf failed, err: %v\n", err)
		return
	}
	fmt.Printf("get conf from etcd success, %v\n", logEntryConf)

	// 派一个哨兵区见识日志收集项的变化(有变化即使通知logAgent实现热加载配置)
	for index, value := range logEntryConf {
		fmt.Printf("index:%v value:%v\n", index, value)
	}

	// 3. 收集日志发往kafka
	taillog.Init(logEntryConf)
	fmt.Println("kafka1")
	newConfChan := taillog.NewConfChan() // 从taillog保重获取对外暴露的通道
	fmt.Println("kafka2")
	var wg sync.WaitGroup
	wg.Add(1)
	go etcd.WatchConf(etcdConfKey, newConfChan) //哨兵发现最新的配置信息会通知上面的那个通道
	fmt.Println("kafka3")
	wg.Wait()

	// run()

}
