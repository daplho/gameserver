package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	go doSignal()

	port := flag.String("port", "7654", "The port to listen to udp traffic on")
	flag.Parse()

	log.Print("Starting Health Ping")
	stop := make(chan struct{})
	go doHealth(stop)

	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start udp server: %v", err)
	}
	defer conn.Close()

	readWriteLoop(conn, stop)
}

func doHealth(stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-stop:
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}

// doSignal shutsdown on SIGTERM/SIGKILL
func doSignal() {
	stop := newStopChannel()
	<-stop
	log.Println("Exit signal received. Shutting down.")
	os.Exit(0)
}

func newStopChannel() <-chan struct{} {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
	}()

	return stop
}

func readPacket(conn net.PacketConn, b []byte) (net.Addr, string) {
	n, sender, err := conn.ReadFrom(b)
	if err != nil {
		log.Fatalf("Could not read from udp stream: %v", err)
	}
	txt := strings.TrimSpace(string(b[:n]))
	log.Printf("Received packet from %v: %v", sender.String(), txt)
	return sender, txt
}

func readWriteLoop(conn net.PacketConn, stop chan struct{}) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)
		parts := strings.Split(strings.TrimSpace(txt), " ")

		switch parts[0] {
		case "EXIT":
			respond(conn, sender, "ACK: "+txt+"\n")
			log.Printf("Received EXIT command. Exiting.")
			os.Exit(0)
		case "UNHEALTHY":
			close(stop)
		}
	}
}

func respond(conn net.PacketConn, sender net.Addr, txt string) {
	if _, err := conn.WriteTo([]byte(txt), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
}
