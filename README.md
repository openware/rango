# RANGO

Rango is a general purpose websocket server which dispatch public and private messages.
It's using AMQP (RabbitMQ) as source of messages.

Rango is made as a drop-in replacement of ranger built in ruby.

## Build
```bash
go build ./cmd/rango
```

## Start the server
```bash
./rango
```

## Connect to public channel
```bash
wscat --connect localhost:8080/public
```

## Connect to private channel
```bash
wscat --connect localhost:8080/private --header "Authorization: Bearer $(go run ./tools/jwt)"
```

## Messages
### Subscribe to a stream list
```
{"event":"subscribe","streams":["eurusd.trades","eurusd.ob-inc"]}
```

### Unsubscribe to one or several streams
```
{"event":"unsubscribe","streams":["eurusd.trades"]}
```
