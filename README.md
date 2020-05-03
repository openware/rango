# RANGO

Rango is a general purpose websocket server which dispatches public and private messages.
It's using AMQP (RabbitMQ) as source of messages.

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

## RPC Methods

Every request and responses are formated in a array the following way:

```json
[0, 42, "method", ["any", "arguments", 51]]
```

- The first argument is 0 for requests and 1 for responses.

- The second argument is the request ID, the client can set the value he wants and the server will include it in the response. This helps to keep track of which response stands for which request. We use often 42 in our examples, you need to change this value by your own according to your implementation.

- Third is the method name

- Last is a list of arguments for the method

### Subscribe to public streams

Request:

```
[0,42,"subscribe",["public",["eurusd.trades","eurusd.ob-inc"]]]
[0,43,"subscribe",["private",["trades","orders"]]]
```

Response:

```
[1,42,"subscribed",["public",["eurusd.trades","eurusd.ob-inc"]]]
[1,43,"subscribed",["private",["trades","orders"]]]
```

### Unsubscribe to one or several streams

Request:

```
[0,42,"unsubscribe",["public",["eurusd.trades","eurusd.ob-inc"]]]
```

Response:

```
[1,42,"unsubscribe",["public",["eurusd.trades","eurusd.ob-inc"]]]
```

## RPC Responses

### Authentication notification
