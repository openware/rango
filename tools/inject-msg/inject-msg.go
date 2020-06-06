package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openware/rango/pkg/upstream"
	"github.com/rs/zerolog/log"
)

var (
	ex       = flag.String("exchange", "peatio.events.ranger", "Exchange name of upstream messages")
	amqpAddr = flag.String("amqp-addr", "amqp://localhost:5672", "AMQP server address")
	wait     = flag.Float64("wait", 2, "Time to wait between submit batch of messages")
)

func main() {
	flag.Parse()

	mq, err := upstream.NewAMQPSession(*amqpAddr)

	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	for {
		file, err := os.Open("msg.txt")
		if err != nil {
			panic(err.Error())
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				panic(err.Error())
			}

			msg := strings.Split(scanner.Text(), " ")

			if err := mq.Push(*ex, msg[0], []byte(msg[1])); err != nil {
				fmt.Printf("Push failed: %s\n", err)
			} else {
				log.Info().Msgf("Pushed on %s msg: %s", msg[0], msg[1])
			}

		}
		file.Close()
		log.Info().Msgf("Waiting %f seconds", *wait)
		time.Sleep(time.Duration(float64(time.Second) * *wait))
	}
}
