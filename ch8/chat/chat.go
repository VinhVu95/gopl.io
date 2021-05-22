// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 254.
//!+

// Chat is a server that lets clients chat with each other.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

//!+broadcaster
type client chan<- string // an outgoing message channel

type clientChannel struct {
	channel    client
	clientName string
}

var (
	entering = make(chan clientChannel)
	leaving  = make(chan clientChannel)
	messages = make(chan string) // all incoming client messages
	clients  = make(map[clientChannel]bool)
)

func broadcaster() {
	//clients := make(map[clientChannel]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				cli.channel <- msg
			}

		case cli := <-entering:
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli.channel)
		}
	}
}

//!-broadcaster

//!+handleConn
func handleConn(conn net.Conn) {
	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	newClient := clientChannel{
		channel:    ch,
		clientName: who,
	}

	broadcastAllClientNames(ch)
	ch <- "You are " + who

	messages <- who + " has arrived"
	entering <- newClient

	shout := make(chan string)

	go checkIdle(shout, conn)

	scan(who, conn, shout)
	// NOTE: ignoring potential errors from input.Err()

	leaving <- newClient
	messages <- who + " has left"
	conn.Close()
}

func broadcastAllClientNames(ch chan string) {
	var clientNames []string
	for cli := range clients {
		clientNames = append(clientNames, cli.clientName)
	}
	ch <- "Current set of clients: " + strings.Join(clientNames, " ")
}

func checkIdle(shout chan string, c net.Conn) {
	ticker := time.NewTicker(1 * time.Second)
	for countdown := 10; countdown > 0; countdown-- {
		fmt.Println(countdown)
		select {
		case <-ticker.C:
		case _ = <-shout:
			countdown = 10
		}
	}
	c.Close()
	ticker.Stop()
}

func scan(who string, conn net.Conn, shout chan string) {
	input := bufio.NewScanner(conn)
	for input.Scan() {
		messages <- who + ": " + input.Text()
		shout <- input.Text()
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

//!-handleConn

//!+main
func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
}

//!-main
