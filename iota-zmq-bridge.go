package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"flag"
	"strconv"
    "time"

	zmq "github.com/pebbe/zmq4"
	iotagox "github.com/iotaledger/iota.go/v2/x"
	iotago "github.com/iotaledger/iota.go/v2"
)

type ZMQServer struct {
    ctx        *zmq.Context
    mq         *zmq.Socket
}

// Global variable to hold the list of indexes to publish
var indexesToPublish []string
var verbosePublishedMessages bool
var verboseErr error

func NewZMQServer(zmqIP string, zmqPort string) (*ZMQServer, error) {
    ctx, err := zmq.NewContext()
    if err != nil {
        return nil, err
    }

    mq, err := ctx.NewSocket(zmq.PUB)
    if err != nil {
        return nil, err
    }

	bindStr := fmt.Sprintf("tcp://%s:%s", zmqIP, zmqPort)
	err = mq.Bind(bindStr)
    if err != nil {
        return nil, err
    }

    return &ZMQServer{
        ctx:         ctx,
        mq:          mq,
    }, nil
}

func (s *ZMQServer) PublishIndexationMessage(topic string, msg *iotago.Message) {

    _, ok := msg.Payload.(*iotago.Indexation)
    if !ok {
        log.Printf("Message payload is not an indexation payload")
        return
    }

    jmsg, err := IndexationMessageToJson(msg)
    if err != nil {
        log.Printf("Error converting message to JSON: %v", err)
        return
    }

	sentMsg := fmt.Sprintf("%s %s", topic, string(jmsg))
    _, err = s.mq.Send(sentMsg, 0)
    if err != nil {
        log.Printf("Error sending message: %v", err)
        return
    }

	if verbosePublishedMessages {
		fmt.Printf("Verbose log: Sent Message to Topic [%s]:\n%s\n", topic, string(jmsg))
	}
}

func IndexationMessageToJson(msg *iotago.Message) ([]byte, error) {
	var jmsg []byte
	var indexationPayload = msg.Payload.(*iotago.Indexation)

	parents := make([]string, len(msg.Parents))
	for index, parentID := range msg.Parents {
		parents[index] = string(hex.EncodeToString(parentID[:]))
	}
	tmsg := map[string]interface{}{
		"id": iotago.MessageIDToHexString(msg.MustID()),
		"networkId": msg.NetworkID,
		"nonce": msg.Nonce,
		"parentMessageIds": parents, 
		"payload": map[string]string{
			"index": string(indexationPayload.Index),
			"data": string(indexationPayload.Data),
		},
	}

	jmsg, err := json.Marshal(tmsg)
	if err != nil {
		return nil, err
	}
	return jmsg, nil
}

func contains(slice []string, str string) bool {
    for _, v := range slice {
        if v == str {
            return true
        }
    }
    return false
}

func shouldPublish(index string) bool {

    if len(indexesToPublish) == 0 {
        return true
    }

    for _, pattern := range indexesToPublish {
        if strings.HasSuffix(pattern, "*") {
            trimmedPattern := strings.TrimSuffix(pattern, "*")
            if strings.HasPrefix(index, trimmedPattern) {
                return true
            }
        } else {
            if index == pattern {
                return true
            }
        }
    }
    return false
}


func worker(server *ZMQServer, messageChannel <-chan *iotago.Message, done chan<- bool) {
    for msg := range messageChannel {
        if indexationPayload, ok := msg.Payload.(*iotago.Indexation); ok {
            topic := string(indexationPayload.Index)
            if shouldPublish(topic) {
                server.PublishIndexationMessage(topic, msg)
            }
        }
    }
    done <- true
}

func connectToBroker(client *iotagox.NodeEventAPIClient, mqttctx context.Context) {
    for {
        if err := client.Connect(mqttctx); err != nil {
            log.Println("Error connecting to IOTA MQTT broker:", err)
            log.Println("Retrying in 5 seconds...")
            time.Sleep(5 * time.Second)
        } else {
            log.Println("Connected to IOTA MQTT broker.")
            break
        }
    }
}

func main() {
    var (
        mqttIPFlag   = flag.String("mqttip", "", "MQTT broker IP")
        mqttPortFlag = flag.String("mqttport", "", "MQTT broker port")
        zmqIPFlag    = flag.String("zmqip", "", "ZMQ server IP to bind to")
        zmqPortFlag  = flag.String("zmqport", "", "ZMQ server port to bind to")
        indexesFlag  = flag.String("indexes", "", "Comma-separated list of message indexes to publish")
        verboseFlag  = flag.Bool("verbose", false, "Enable verbose logging")
        bufferSizeFlag = flag.Int("buffersize", 0, "Buffer size for incoming MQTT messages")
        numWorkersFlag = flag.Int("numworkers", 0, "Number of workers processing messages")
    )

    flag.Parse()

    log.Println("Setting environment configuration...")

    zmqIP := *zmqIPFlag
    if zmqIP == "" {
        zmqIP = os.Getenv("ZMQ_IP")
        if zmqIP == "" {
            zmqIP = "0.0.0.0"
        }
    }
    log.Printf("ZMQ_IP: %s\n", zmqIP)

    zmqPort := *zmqPortFlag
    if zmqPort == "" {
        zmqPort = os.Getenv("ZMQ_PORT")
        if zmqPort == "" {
            zmqPort = "5556"
        }
    }
    log.Printf("ZMQ_PORT: %s\n", zmqPort)

    mqttIP := *mqttIPFlag
    if mqttIP == "" {
        mqttIP = os.Getenv("MQTT_IP")
        if mqttIP == "" {
            mqttIP = "localhost"
        }
    }
    log.Printf("MQTT_IP: %s\n", mqttIP)

    mqttPort := *mqttPortFlag
    if mqttPort == "" {
        mqttPort = os.Getenv("MQTT_PORT")
        if mqttPort == "" {
            mqttPort = "1883"
        }
    }
    log.Printf("MQTT_PORT: %s\n", mqttPort)


    // Check command line for indexes first
    if *indexesFlag != "" {
        indexesToPublish = strings.Split(*indexesFlag, ",")
    } else {
        // If not available in command line, get from environment variable
        iotaMessageIndexesToPublishEnv := os.Getenv("INDEXES")
        if iotaMessageIndexesToPublishEnv != "" {
            indexesToPublish = strings.Split(iotaMessageIndexesToPublishEnv, ",")
        }
    }
    if len(indexesToPublish) == 0 {
        log.Printf("INDEXES: [ALL]\n")
    } else {
        log.Printf("INDEXES: %s\n", indexesToPublish)
    }

    bufferSize := *bufferSizeFlag
    if bufferSize <= 0 {
        bufferSizeEnv := os.Getenv("BUFFER_SIZE")
        if bufferSizeEnv != "" {
            bufferSizeFromEnv, err := strconv.Atoi(bufferSizeEnv)
            if err == nil && bufferSizeFromEnv > 0 {
                bufferSize = bufferSizeFromEnv
            }
        }
    }
    if bufferSize <= 0 {
        bufferSize = 1000
    }
    log.Printf("BUFFER_SIZE: %d\n", bufferSize)

    numWorkers := *numWorkersFlag
    if numWorkers <= 0 {
        numWorkersEnv := os.Getenv("NUM_WORKERS")
        if numWorkersEnv != "" {
            numWorkersFromEnv, err := strconv.Atoi(numWorkersEnv)
            if err == nil && numWorkersFromEnv > 0 {
                numWorkers = numWorkersFromEnv
            }
        }
    }
    if numWorkers <= 0 {
        numWorkers = 10
    }
    log.Printf("NUM_WORKERS: %d\n", numWorkers)

    verbosePublishedMessages = *verboseFlag
    if !verbosePublishedMessages { // if not set by the flag
        verbosePublishedMessagesEnv := os.Getenv("VERBOSE_MESSAGES")
        verbosePublishedMessages, verboseErr = strconv.ParseBool(verbosePublishedMessagesEnv)
        if verboseErr != nil {
            verbosePublishedMessages = false
        }
    }
    log.Printf("VERBOSE_MESSAGES: %v\n", verbosePublishedMessages)

    // Create ZMQ server
    server, err := NewZMQServer(zmqIP, zmqPort)
    if err != nil {
        log.Fatalf("Error creating ZMQ server: %v", err)
    }

    defer server.mq.Close()
    defer server.ctx.Term()

    // Form the complete MQTT broker URI by combining address and port
    brokerURI := fmt.Sprintf("mqtt://%s:%s", mqttIP, mqttPort)

    // Create an IOTA NodeAPI client to connect to the MQTT broker
    iotaClient := iotagox.NewNodeEventAPIClient(brokerURI)

    // Connect to the MQTT broker with retries until successful
    mqttctx := context.Background()
    connectToBroker(iotaClient, mqttctx)

    // Subscribe to the messages channel with buffering
    messageChan := make(chan *iotago.Message, bufferSize)
    for i := 0; i < numWorkers; i++ {
        go func() {
            for msg := range messageChan {
                if indexationPayload, ok := msg.Payload.(*iotago.Indexation); ok {
                    topic := string(indexationPayload.Index)
                    if shouldPublish(topic) {
                        server.PublishIndexationMessage(topic, msg)
                    }
                }
            }
        }()
    }

    // Set up a signal handler to catch termination signals (Ctrl+C)
    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

    reconnect := false

    for {
        if reconnect {
            // Close previous connection if any
            iotaClient.Close()
            
            // Wait for 5 seconds before attempting to reconnect
            time.Sleep(5 * time.Second)
            
            if err := iotaClient.Connect(mqttctx); err != nil {
                log.Println("Error: reconnecting to IOTA MQTT broker:", err)
                continue
            } else {
                log.Println("Reconnected to IOTA MQTT broker.")
                reconnect = false
            }
        }
        
        select {
        case msg := <-iotaClient.Messages():
            // Push to the buffer
            messageChan <- msg
        case err := <-iotaClient.Errors:
            log.Println("Error:", err)
            reconnect = true
            log.Println("Trying to reconnect to IOTA MQTT broker...")
        case <-signalCh:
            // Handle Ctrl+C or other termination signals
            log.Println("Received termination signal. Exiting...")
            iotaClient.Close()
            server.mq.Close()
            server.ctx.Term()
            close(messageChan) 
            return
        }
    }

}



