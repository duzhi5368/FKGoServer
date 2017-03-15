//---------------------------------------------
package Kafka
//---------------------------------------------
import (
	LOG "log"

	CLI "gopkg.in/urfave/cli.v2"
	SARAME "github.com/Shopify/sarama"
)
//---------------------------------------------
var (
	kAsyncProducer SARAME.AsyncProducer
	kClient        SARAME.Client
	ChatTopic      string
)
//---------------------------------------------
func initKafka(c *CLI.Context) {
	addrs := c.StringSlice("kafka-brokers")
	ChatTopic = c.String("chat-topic")
	config := SARAME.NewConfig()
	config.Producer.Return.Successes = false
	config.Producer.Return.Errors = false
	producer, err := SARAME.NewAsyncProducer(addrs, config)
	if err != nil {
		LOG.Fatalln(err)
	}

	kAsyncProducer = producer
	cli, err := SARAME.NewClient(addrs, nil)
	if err != nil {
		LOG.Fatalln(err)
	}
	kClient = cli
}
//---------------------------------------------
func Init(c *CLI.Context) {
	initKafka(c)
}
//---------------------------------------------
func NewConsumer() (SARAME.Consumer, error) {
	return SARAME.NewConsumerFromClient(kClient)
}
//---------------------------------------------