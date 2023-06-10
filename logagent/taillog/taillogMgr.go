package taillog

import (
	"fmt"
	"logagent/etcd"
	
	"time"
)

var tskMgr *tailLogMgr

type tailLogMgr struct {
	logEntry    []*etcd.LogEntry
	tskMap      map[string]*TailTask
	newConfChan chan []*etcd.LogEntry
}

func Init(logEntryConf []*etcd.LogEntry) {
	tskMgr = &tailLogMgr{
		logEntry:    logEntryConf,
		tskMap:      make(map[string]*TailTask),
		newConfChan: make(chan []*etcd.LogEntry), //无缓冲的通道
	}
	for _, logEntry := range logEntryConf {
		// conf = etcd.LogEntry
		//
		tailObj := NewTailTask(logEntry.Path, logEntry.Topic)
		mk := fmt.Sprintf("%s_%s", logEntry.Path, logEntry.Topic)
		tskMgr.tskMap[mk] = tailObj
	}
	go tskMgr.run()
}

// 监听自己的newConfChan, 有了新的配置过来之后就做对应的处理
// 1. 配置新增
// 2. 配置删除
// 3. 配置变更
func (t *tailLogMgr) run() {
	for {
		select {
		case newConf := <-t.newConfChan:
			fmt.Println("新的配置来了", newConf)

			for _, conf := range newConf {
				mk := fmt.Sprintf("%s_%s", conf.Path, conf.Topic)
				_, ok := t.tskMap[mk]
				if ok {
					continue
				} else {
					// 新增的
					tailObj := NewTailTask(conf.Path, conf.Topic)
					t.tskMap[mk] = tailObj
				}
			}
			for _, c1 := range t.logEntry {
				isDelete := true
				for _, c2 := range newConf {
					if c2.Path == c1.Path && c2.Topic == c1.Topic {
						isDelete = false
						continue
					}
					if isDelete {
						// 把c1对应的这个taillObj给停掉
						mk := fmt.Sprintf("%s_%s", c1.Path, c1.Topic)
						// t.skkMap[mk] ==> tailObj
						t.tskMap[mk].cancelFunc()
					}
				}
			}
		default:
			time.Sleep(time.Second)
		}
	}
}

// 向外暴露tskMgr的newConfChan

func NewConfChan() chan<- []*etcd.LogEntry {
	return tskMgr.newConfChan
}
