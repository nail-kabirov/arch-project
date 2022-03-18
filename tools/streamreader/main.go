package main

import (
	"bytes"
	"container/ring"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

const consumerName = "test"
const consumerStopDuration = time.Millisecond * 500

var countFlag = flag.Int("count", 0, "last message count")

func main() {
	flag.Parse()
	if countFlag == nil || *countFlag <= 0 {
		flag.Usage()
		os.Exit(0)
	}

	cfg, err := parseEnv()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	env, err := initRabbitMQEnv(cfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	buffer := ring.New(*countFlag)
	msgCount, err := calculateTotalMessageCount(env, cfg.StreamName, &buffer)
	if err != nil {
		fmt.Printf("create consumer error - %v", err)
		os.Exit(1)
	}

	fmt.Printf("total message count - %v\n", msgCount)

	buffer.Do(func(val interface{}) {
		if val != nil {
			data := val.(msgData)
			msg := data.msg
			var buf bytes.Buffer
			if err := json.Indent(&buf, data.msg, "", "  "); err == nil {
				msg = buf.Bytes()
			}
			fmt.Printf("%v: %v\n", data.index, string(msg))
		}
	})
}

func initRabbitMQEnv(cfg *config) (*stream.Environment, error) {
	port, err := strconv.Atoi(cfg.RMQPort)
	if err != nil {
		return nil, err
	}
	return stream.NewEnvironment(
		stream.NewEnvironmentOptions().
			SetHost(cfg.RMQHost).
			SetPort(port).
			SetUser(cfg.RMQUser).
			SetPassword(cfg.RMQPassword).
			SetAddressResolver(stream.AddressResolver{Host: cfg.RMQHost, Port: port}),
	)
}

func calculateTotalMessageCount(env *stream.Environment, streamName string, buffer **ring.Ring) (int64, error) {
	count := int64(0)

	options := stream.NewConsumerOptions().
		SetConsumerName(consumerName).
		SetCRCCheck(true).
		SetOffset(stream.OffsetSpecification{}.First()).
		SetManualCommit()

	var timer *time.Timer

	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		if timer != nil {
			timer.Reset(consumerStopDuration)
		}
		if len(message.Data) == 1 {
			(*buffer).Value = msgData{
				index: count,
				msg:   message.Data[0],
			}
			*buffer = (*buffer).Next()
			count++
		}
	}

	consumer, err := env.NewConsumer(streamName, handleMessages, options)
	if err != nil {
		return 0, err
	}
	timer = time.NewTimer(consumerStopDuration)

	defer consumer.Close()

	<-timer.C

	return count, nil
}

type msgData struct {
	index int64
	msg   []byte
}
