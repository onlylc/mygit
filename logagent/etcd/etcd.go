package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

// etcd client put/get demo
// use etcd/clientv3

var (
	cli *clientv3.Client
)

type LogEntry struct {
	Path  string `json:"path"`
	Topic string `json:"topic"`
}

func Init(addr string, timeout time.Duration) (err error) {
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{addr},
		DialTimeout: timeout,
	})
	if err != nil {
		// handle error!
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		return
	}

	return
	// defer cli.Close()
	// // put
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// value := `[{"path":"c:/tmp/nginx.log",""topic":"web_log"}]`
	// _, err = cli.Put(ctx, "/logagent/127.0.0.1/collect_config", value)
	// cancel()
	// if err != nil {
	// 	fmt.Printf("put to etcd failed, err:%v\n", err)
	// 	return
	// }
	// // get
	// ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	// resp, err := cli.Get(ctx, "q1mi")
	// cancel()
	// if err != nil {
	// 	fmt.Printf("get from etcd failed, err:%v\n", err)
	// 	return
	// }
	// for _, ev := range resp.Kvs {
	// 	fmt.Printf("%s:%s\n", ev.Key, ev.Value)
	// }
}

// 2.1 从etcd中获取日志收集项的配置信息
func GetConf(key string) (logEntryConf []*LogEntry, err error) {
	// get
	fmt.Println(key)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := cli.Get(ctx, "/logagent/collect_config")
	fmt.Println("key", resp.Kvs)
	cancel()
	if err != nil {
		fmt.Printf("get from etcd failed, err:%v\n", err)
		return
	}
	for _, ev := range resp.Kvs {

		err = json.Unmarshal(ev.Value, &logEntryConf)
		fmt.Println("etcd", logEntryConf)
		if err != nil {
			fmt.Printf("Unmarshal etcd value  failed, err:%v\n", err)
			return
		}

	}
	return
}

func WatchConf(key string, newConfCh chan<- []*LogEntry) {
	ch := cli.Watch(context.Background(), key)
	// 从通道尝试取值(监视的信息)
	for wresp := range ch {
		for _, evt := range wresp.Events {
			// 通知别人
			// 1. 先判段操作的类型
			var newConf []*LogEntry
			if evt.Type != clientv3.EventTypeDelete {
				// 如果是删除操作
				fmt.Println(evt)
				err := json.Unmarshal(evt.Kv.Value, &newConf)
				if err != nil {
					fmt.Printf("unmarshal failed, err:%v\n", err)
					continue
				}
				fmt.Printf("get new conf:%v\n", &newConf)
			}

			newConfCh <- newConf
		}
	}
}
