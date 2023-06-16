package main

import (
	"flag"
	"fmt"

	"github.com/openware/pkg/jwt"
)

var (
	port  = flag.String("port", "7070", "Port to bind")
	uid   = flag.String("uid", "IDABC0000001", "UID")
	email = flag.String("email", "admin@barong.io", "Email")
	role  = flag.String("role", "admin", "Role")
	level = flag.Int("level", 3, "Level")
	ref   = flag.Int("ref", 1, "Referral id")
)

func main() {
	flag.Parse()

	ks, _ := jwt.LoadOrGenerateKeysEdDSA("config/ed25519-key", "config/ed25519-key.pub")
	t, _ := jwt.ForgeTokenEdDSA(*uid, *email, *role, *level, *ref, ks.PrivateKey, nil)
	fmt.Print(t)
}
