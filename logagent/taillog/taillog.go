package taillog

import (
	"context"
	"fmt"
	"logagent/kafka"

	"github.com/hpcloud/tail"
)

// logAgent的工作流程
// 1. 读日志  tailf 第三方库
// 2. 往kafka写日志

// var (
// 	tailObj *tail.Tail
// 	logChan chan string
// )

type TailTask struct {
	path     string
	topic    string
	instance *tail.Tail
	// 为了能实现退出t.run()
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewTailTask(path, topic string) (tailObj *TailTask) {
	ctx, cancel := context.WithCancel(context.Background())
	tailObj = &TailTask{
		path:       path,
		topic:      topic,
		ctx:        ctx,
		cancelFunc: cancel,
	}
	tailObj.init()

	return
}

func (t *TailTask) init() {
	config := tail.Config{
		ReOpen:    true,                                 // 重新打开
		Follow:    true,                                 // 是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // 从文件的那个地方开始读
		MustExist: false,                                // 是否存在日志文件, 不存在不报错
		Poll:      true,                                 //
	}
	var err error
	t.instance, err = tail.TailFile(t.path, config)
	if err != nil {
		fmt.Println("tail file failed, err:", err)
		return
	}
	go t.run()
}

func (t *TailTask) run() {
	for {
		select {
		case <-t.ctx.Done():
			fmt.Printf("tail task:%s_%s 结束了...\n", t.path, t.topic)
			return
		case line := <-t.instance.Lines:
			// kafka.SendToKafka(t.topic, line.Text)
			// 先把日志数据发送到一个通道中
			kafka.SendToChan(t.topic, line.Text)
			// kafka那个包中有单独的goroutine区取日志数据发送到kafka
		}
	}
}

// func (t *TailTask) ReadChan() <-chan *tail.Line {
// 	return t.instance.Lines
// }
