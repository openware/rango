package main

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"math/rand"

	"github.com/rs/zerolog/log"

	"github.com/openware/rango/pkg/auth"
	"github.com/openware/rango/pkg/routing"
	"github.com/openware/rango/pkg/upstream"
)

var (
	wsAddr   = flag.String("ws-addr", ":8080", "http service address")
	amqpAddr = flag.String("amqp-addr", "amqp://localhost:5672", "AMQP server address")
	pubKey   = flag.String("pubKey", "config/rsa-key.pub", "Path to public key")
	ex       = flag.String("exchange", "peatio.events.ranger", "Exchange name of upstream messages")
)

const prefix = "Bearer "

type httpHanlder func(w http.ResponseWriter, r *http.Request)

func token(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(string(authHeader), prefix) {
		return ""
	}

	return authHeader[len(prefix):]
}

func authHandler(h httpHanlder, key *rsa.PublicKey, mustAuth bool) httpHanlder {
	return func(w http.ResponseWriter, r *http.Request) {
		auth, err := auth.ParseAndValidate(token(r), key)

		if err != nil && mustAuth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if err == nil {
			r.Header.Set("JwtUID", auth.UID)
		} else {
			r.Header.Del("JwtUID")
		}
		h(w, r)
		return
	}
}

func main() {
	flag.Parse()
	hub := routing.NewHub()

	ks := auth.KeyStore{}
	ks.GenerateKeys()
	if err := ks.LoadPublicKey(*pubKey); err != nil {
		log.Fatal().Msg("LoadPublicKey failed: " + err.Error())
		return
	}

	rand.Seed(time.Now().UnixNano())
	qName := fmt.Sprintf("rango.instance.%d", rand.Int())
	ach, err := upstream.NewAMQPSession(*amqpAddr).Stream(*ex, qName)

	go hub.ListenWebsocketEvents()
	go hub.ListenAMQP(ach)

	if err != nil {
		log.Fatal().Msgf("AMQP init failed: %s", err.Error())
		return
	}

	wsHandler := func(w http.ResponseWriter, r *http.Request) {
		routing.NewClient(hub, w, r)
	}

	http.HandleFunc("/private", authHandler(wsHandler, ks.PublicKey, true))
	http.HandleFunc("/public", authHandler(wsHandler, ks.PublicKey, false))
	http.HandleFunc("/", authHandler(wsHandler, ks.PublicKey, false))

	log.Printf("Listenning on %s", *wsAddr)
	err = http.ListenAndServe(*wsAddr, nil)
	if err != nil {
		log.Fatal().Msg("ListenAndServe failed: " + err.Error())
	}
}
