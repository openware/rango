package upstream

import "time"

type Msg struct {
	Channel string
	Message string
}

func AMQPUpstream(output chan Msg) {
	msg := Msg{"public_trades", "0.1,0.12"}
	for {
		time.Sleep(2 * time.Second)
		output <- msg
	}
}
