package kafka

import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
)

type logData struct {
	topic string
	data  string
}

// 基于sarama第三方库开发的kafka client
var (
	client      sarama.SyncProducer //声明一个全局的链接kafka的生产者
	logDataChan chan *logData
)

func Init(addrs []string, maxSize int) (err error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll          // 发送完数据需要leader和follow都确认
	config.Producer.Partitioner = sarama.NewRandomPartitioner // 新选出一个partition
	config.Producer.Return.Successes = true                   // 成功交付的消息将在success channel返回

	// // 构造一个消息

	// 连接kafka
	client, err = sarama.NewSyncProducer(addrs, config)
	if err != nil {
		fmt.Println("producer closed, err:", err)
		return
	}
	// defer client.Close()
	// // 发送消息

	logDataChan = make(chan *logData, maxSize)
	go sendToKafka()
	return
}

func SendToChan(topic, data string) {
	msg := &logData{
		topic: topic,
		data:  data,
	}
	logDataChan <- msg
}

func sendToKafka() {
	for {
		select {
		case lg := <-logDataChan:
			msg := &sarama.ProducerMessage{}
			msg.Topic = lg.topic
			msg.Value = sarama.StringEncoder(lg.data)
			pid, offset, err := client.SendMessage(msg)
			if err != nil {
				fmt.Println("send msg failed, err:", err)
				return
			}
			fmt.Printf("pid:%v offset:%v\n", pid, offset)
		default:
			time.Sleep(time.Millisecond * 50)
		}

	}

}
