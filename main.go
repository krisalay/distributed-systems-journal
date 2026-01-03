package main

import (
	"fmt"

	"github.com/krisalay/distributed-systems-journal/hashring"
)

func main() {
	r := hashring.New()
	r.AddNode("A")
	r.AddNode("B")
	r.AddNode("C")

	for _, k := range []string{"user:1", "user:2", "user:3"} {
		fmt.Println(k, "->", r.GetNode(k))
		fmt.Println(k, "replicas ->", r.GetNodes(k, 2))
	}
}
