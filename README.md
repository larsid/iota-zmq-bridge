# [larsid/iota-zmq-bridge:1.0.0](https://hub.docker.com/layers/larsid/iota-zmq-bridge/1.0.0/images/sha256-87f34d9945042b0448fda04501e912ca55f32a069a547d4a59e6c24b81046abc?context=repo)

# IOTA ZMQ Bridge Server

**Version:** 1.0.0

`IOTA ZMQ Bridge` is a specific publish-subscriber ZMQ server designed to bridges between an IOTA Tangle HORNET node (v1.2.x) with MQTT broker and a ZeroMQ (ZMQ) server. It allows ZMQ subscriber clients to receive specific IOTA messages with indexation payload from an IOTA Tangle distributed ledger network.

## Table of Contents

- [Introduction](#Introduction)
  - [ZMQ Client Exemple](#ZMQ-Client-Exemple)
  - [The Response Message Format](#The-Response-Message-Format)
- [Features](#features)
- [Dependencies](#dependencies)
- [Setup & Usage](#setup--usage)
  - [Prerequisites](#prerequisites)
  - [Installing the Software](#installing-the-software)
  - [Running the ZMQ Bridge Server](#running-the-ZMQ-bridge-server)
  - [Command-Line Options](#command-line-options)
  - [Environment Variables](#environment-variables)
- [Building a Docker Container Image](#Building-a-Docker-Container-Image)
- [Creating and Running a Docker Container](#Creating-and-Running-a-Docker-Container)
- [License](#license)
- [Contributions](#contributions)

## Introduction 

The provided software implements a ZMQ publish-subscriber bridge server that relays IOTA Tangle network messages based on the indexation payload to ZMQ subscribed clients. It can seamlessly listen to MQTT messages from a IOTA Tangle HORNET node (v1.2.x), filter messages with indexation payloads based on specified indexes, and publish the relevant messages to the ZMQ server. If a client wishes to subscribe to these messages based on a specific index, they can use a ZMQ client library to connect to the ZMQ server and listen for the desired topics (indexes).

### ZMQ Client Exemple

This solution acts as a bridge, and its primary function is to relay specific IOTA messages from the MQTT broker to ZMQ clients. To accomplish this, a ZMQ client should:

- Connect to the IOTA ZMQ Bridge server using the specified IP and port;
- Subscribe to a specific topic or multiple topics.

Subsequently, the IOTA ZMQ Bridge server will use the topic string to filter IOTA MQTT messages by index and forward the indexed messages to the subscribed clients. Here's a basic example of how a client can subscribe to different topics (indexes) using the ZMQ library in Go:

```powershell
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	zmq "github.com/pebbe/zmq4"
)

func main() {

	// Command-line arguments with default values
	ip := flag.String("ip", "localhost", "IP address of the ZMQ server")
	port := flag.String("port", "5556", "Port of the ZMQ server")
	topicStr := flag.String("topics", "INDEX_A,INDEX_B,INDEX_C", "Comma-separated list of topics to subscribe to")

	flag.Parse()

	// Convert comma-separated topics to a slice
	topics := strings.Split(*topicStr, ",")

	// Create a new ZeroMQ SUB socket
	subscriber, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatalf("Failed to create socket: %s", err)
	}
	defer subscriber.Close()

	// Connect to the ZMQ server using the provided IP and Port
	connectionString := fmt.Sprintf("tcp://%s:%s", *ip, *port)
	err = subscriber.Connect(connectionString)
	if err != nil {
		log.Fatalf("Failed to connect: %s", err)
	}

	// Subscribe to the desired topics
	for _, topic := range topics {
		err = subscriber.SetSubscribe(topic)
		if err != nil {
			log.Fatalf("Failed to subscribe to topic %s: %s", topic, err)
		}
		fmt.Printf("Subscribed to topic: %s\n", topic)
	}

	for {
		// Receive message from server
		message, err := subscriber.Recv(0)
		if err != nil {
			log.Printf("Failed to receive message: %s", err)
			continue
		}
	
		// Split the message at the first space to get topic and JSON payload
		parts := strings.SplitN(message, " ", 2)
		if len(parts) == 2 {
			topic := parts[0]
			jsonPayload := parts[1]
			fmt.Printf("Received message topic [%s] payload: %s\n", topic, jsonPayload)
		} else {
			log.Printf("Received message with unexpected format: %s", message)
		}
	}
	
}
```

### The Response Message Format

The ZMQ server converts incoming MQTT messages into JSON format before publishing them to ZMQ clients. If the message is of the `Indexation` type, it is passed to the `dexationMessageToJson()` function, which transforms the message into a JSON structure. Below is an example of a response message in JSON string format that the IOTA ZMQ server sends to subscribed clients in response to an incoming indexation "HORNET Spammer" message from the IOTA MQTT broker:

```powershell
{
    "id": "558205d4b04afa2aa0e79c0fbd60ba65e2e56e5eb70b703e1a906ae863bc036c",
    "networkId": 2321005821356827533,
    "nonce": 131365,
    "parentMessageIds": [
        "9d5820690742365f50886a600639fb82b0690fec8bb1f17371f66bc53d3284ff",
        "a8055210fb73449848ad7b6ad87373d8b6a65d7e9be66314f47f7b002d288066",
        "ab4eaebe32676dfb6b9485ed2cf9dd07ab517d53bbe30991bcca1408c64f21d6",
        "d81ff723975190c6ff97c47c3dacfd39e6d5e09927c363f8c0241508221c8a98"
    ],
    "payload": {
        "data": "one-click-tangle.\nCount: 000147\nTimestamp: 2023-10-29T09:46:13Z\nTipselection: 6µs",
        "index": "HORNET Spammer"
    }
}
```

## Features

- **ZMQ Server Integration:** Built-in support for ZeroMQ subscriber connections.
- **IOTA MQTT Broker Connection:** Seamless connection to the IOTA MQTT broker (hornet v1.2.x) to fetch indexation type messages.
- **Message Filtering:** Ability to filter IOTA MQTT Broker messages with indexation payloads based on user-defined indexes. 
- **Optimized Performance:** Designed for high throughput with adjustable buffer size and worker count.
- **Connection Resilience:** If the MQTT broker connection drops, the bridge will attempt to reconnect repeatedly until the server comes back online (waits 5 seconds between each attempt).
- **Verbose Logging:** Configurable logging of indexation messages to assist with troubleshooting and monitoring.

## Dependencies

- `github.com/pebbe/zmq4`: ZeroMQ bindings for Go.
- `github.com/iotaledger/iota.go/v2`: The official Go client library for IOTA.

## Setup & Usage

### Prerequisites
- Ensure that you have Go version 1.14+ installed on your machine.
- Make sure an IOTA MQTT broker is accessible.
- Have ZeroMQ installed on your local machine.

### Installing the Software

To set up the software, follow these steps:

```powershell
go get -u github.com/iotaledger/iota.go/v2
go get -u github.com/iotaledger/iota.go/v2/x
go get -u github.com/pebbe/zmq4
go build -o iota-zmq-bridge
```

### Running the ZMQ Bridge Server

To run the IOTA ZMQ bridge server, you can use the following command example, changing the default parameter values:

```powershell
./iota-zmq-bridge -mqttip="localhost" -mqttport=1883 -zmqip="0.0.0.0" -zmqport=5556 -indexes="" -buffersize=1000 -numworkers=10 -verbose=false
```

### Command-Line Options

Here are the command-line options you can use:

- **mqttip**: Specify the IP of the MQTT broker (default: "localhost").
- **mqttport**: Indicate the Port of the MQTT broker (default: "1883").
- **zmqip**: Determine the IP to bind the ZMQ server (default: "0.0.0.0").
- **zmqport**: Assign the Port to bind the ZMQ server (default: "5556").
- **indexes**: Provide a comma-separated list of message indexes to filter from IOTA MQTT Broker to ZMQ server. Wildcards can be used to specify an index. For example, "LB_*" will filter all messages whose index starts with "LB_". If this option is not specified, all IOTA MQTT Broker indexation messages will be forward to ZMQ server.
- **buffersize**: Specify the buffer size for incoming MQTT messages.
- **numworkers**: Define the number of workers processing messages.
- **verbose**: To enable verbose logging of filtered indexation messages, set this flag.

### Environment Variables

You can also configure the IOTA bridge server using environment variables. If both command-line options and environment variables are provided, the command-line options will take precedence:

- **MQTT_IP**: Set the IP of the MQTT broker.
- **MQTT_PORT**: Specify the Port of the MQTT broker.
- **ZMQ_IP**: Assign the IP to bind the ZMQ server.
- **ZMQ_PORT**: Designate the Port to bind the ZMQ server.
- **INDEXES**: Provide a comma-separated list of message indexes to filter from IOTA MQTT Broker to ZMQ server. Wildcards can be used to specify an index. For example, "LB_*" will filter all messages whose index starts with "LB_". If this option is not specified, all IOTA MQTT Broker indexation messages will be forward to ZMQ server.
- **BUFFER_SIZE**: Specify the buffer size for incoming MQTT messages.
- **NUM_WORKERS**: Define the number of workers processing messages.
- **VERBOSE_MESSAGES**: Set this variable to enable verbose logging of filtered indexation messages.

## Building a Docker Container Image

With the terminal in the image directory, run:

```powershell
docker build -t larsid/iota-zmq-bridge:1.0.0 . 
```
   
## Creating and Running a Docker Container

After building the iota-zmq-bridge image, run:

```powershell
docker run -d -p 5556:5556 --name iota-zmq -e INDEXES="LB_*" -e MQTT_IP="172.16.103.4" larsid/iota-zmq-bridge:1.0.0
```

The example docker run command is provided with `-e` parameter values for MQTT_IP and INDEXES environment variables. You can also use the `-e` parameter to set other environment variables. The parameters you can override are listed below:

| Parameter| Description| Default value|
| ----------------| --------------- | ------------------- |
|MQTT_IP|Set the IP of the MQTT broker.|"localhost"|
|MQTT_PORT|Specify the Port of the MQTT broker.|1883|
|ZMQ_IP|Assign the IP to bind the ZMQ server.|"0.0.0.0"|
|ZMQ_PORT|Designate the Port to bind the ZMQ server.|5556|
|INDEXES|Provide a comma-separated list of message indexes to filter from IOTA MQTT Broker to ZMQ server. Wildcards can be used to specify an index. For example, "LB_*" will filter all messages whose index starts with "LB_". If this option is not specified, all IOTA MQTT Broker indexation messages will be forward to ZMQ server.|"" (All)|
|BUFFER_SIZE|Specify the buffer size for incoming MQTT messages.|1000|
|NUM_WORKERS|Define the number of workers processing messages.|10|
|VERBOSE_MESSAGES|Set this variable to enable verbose logging of filtered indexation messages.|false|

## License

IOTA ZMQ Bridge is open-source software released under the MIT License.

## Contributions

We welcome contributions from the community! If you have any enhancements, bug fixes, or feature requests in mind, feel free to submit a pull request or [create an issue](https://github.com/larsid/dockerfiles/issues). Let's make IOTA ZMQ Bridge even better together!
