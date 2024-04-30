package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/6.5840-dsm/dsm"
)

func main() {
	numpages := 10
	for i, args := range os.Args {
		if args == "-c" {
			clients := make(map[int]string)
			for j := i + 1; j < len(os.Args); j++ {
				clients[j] = os.Args[j]
			}
			dsm.CentralSetup(clients, numpages)
		} else if args == "-p" {

			numservers := 2
			index, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				log.Fatal("could not parse index", err)
			}
			peer := os.Args[i+2]
			central := os.Args[i+3]
			dsm.ClientSetup(numpages, index, numservers, peer, central)
		} else if args == "-h" {
			fmt.Println("If you want to run a central server, use the -c flag followed by the addresses of the clients.")
			fmt.Println("If you want to run a client, use the -p flag followed by the index of the client, the address of the peer, and the address of the central server.")
		}
	}
}
