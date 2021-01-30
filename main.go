package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

var duration int64
var packetSize int

func main() {
	var addr string
	flag.StringVar(&addr, "c", "", "act as client and provide target server adress \nex: -c 127.0.0.1:4040")
	var port string
	flag.StringVar(&port, "s", "", "act as server and provide port to listen on \nex: -s :4040")

	//NOT YET USABLE
	//flag.IntVar(&packetSize, "p", 1e3, "Packetsize in bytes (default is 1 KByte). Has to be the same for server and client in order to work correctly. \nex: -s 1000")
	//flag.Int64Var(&duration, "d", 5, "Speedtest warmup and test duration in seconds for each down- and upload (default is 5). \nex: -d 5")
	//NOT YET USABLE

	flag.Parse()

	//CHANGE WHEN FLAG IS USABLE
	duration = 5
	packetSize = 1e3
	//CHANGE WHEN FLAG IS USABLE

	duration *= 1e3

	if addr != "" && port != "" {
		println("You can only one of the parameters. Either -c or -s but not both.")
		os.Exit(1)
	}
	if addr == "" && port == "" {
		println("You have to use either -c or -s to start as client or as server.")
		os.Exit(1)
	}
	if addr != "" {
		client(addr)
	}
	if port != "" {
		server(port)
	}

}

func client(addr string) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	checkError(err)
	conn, err := net.DialTCP("tcp", nil, raddr)
	checkError(err)
	println("connected to: " + conn.RemoteAddr().String())
	packet := make([]byte, packetSize)
	for i := 0; i < 2; i++ {
		var packetcount uint64 = 0
		startTime := time.Now()
		timerStartTime := startTime
		timeElapsed := time.Since(startTime)
		go func() {
			timerStartTimeSince := time.Since(timerStartTime)
			for timerStartTimeSince.Milliseconds() < duration*2 {
				if i == 0 {
					fmt.Printf("\rTesting download Speed (%ds/"+fmt.Sprint(duration*2/1e3)+"s) | %.2fMbit/s", int(timerStartTimeSince.Seconds()), calcRate(packetcount, timeElapsed))
				} else if i == 1 {
					fmt.Printf("\rTesting upload Speed (%ds/"+fmt.Sprint(duration*2/1e3)+"s) | %.2fMbit/s", int(timerStartTimeSince.Seconds()), calcRate(packetcount, timeElapsed))
				}
				time.Sleep(10e6)
				timerStartTimeSince = time.Since(timerStartTime)
			}
		}()
		var err error
		for timeElapsed.Milliseconds() < duration {
			if i == 0 {
				_, err = io.ReadFull(conn, packet)
			} else if i == 1 {
				_, err = conn.Write(packet)
			}
			if err == nil {
				packetcount++
			} else {
				break
			}
			timeElapsed = time.Since(startTime)
		}
		startTime = time.Now()
		timeElapsed = time.Since(startTime)
		packetcount = 0
		for timeElapsed.Milliseconds() < duration {
			if i == 0 {
				_, err = io.ReadFull(conn, packet)
			} else if i == 1 {
				_, err = conn.Write(packet)
			}

			if err == nil {
				packetcount++
			} else {
				break
			}
			timeElapsed = time.Since(startTime)
		}

		if i == 0 {
			time.Sleep(1e8)
			println()
			println("Average Download Speed: ", fmt.Sprintf("%.2f", calcRate(packetcount, timeElapsed)), " Mbit/s")
		} else if i == 1 {
			println()
			println("Average Upload Speed: ", fmt.Sprintf("%.2f", calcRate(packetcount, timeElapsed)), " Mbit/s")
		}
	}
}

func calcRate(packetCount uint64, timeElapsed time.Duration) float64 {
	return (((float64(packetCount) / (float64(timeElapsed.Nanoseconds()) / 1e9)) / (1e6 / float64(packetSize))) * 8)
}

func server(port string) {
	lport, err := net.ResolveTCPAddr("tcp", port)
	checkError(err)
	l, err := net.ListenTCP("tcp", lport)
	checkError(err)
	defer l.Close()
	println("listening at (tcp) " + lport.String())
	conn, err := l.AcceptTCP()
	checkError(err)
	println("connected to: " + conn.RemoteAddr().String())

	packet := make([]byte, 1000)
	startTime := time.Now()
	timeElapsed := time.Since(startTime)
	for timeElapsed.Milliseconds() < duration*2 {
		conn.Write(packet)
		timeElapsed = time.Since(startTime)
	}
	startTime = time.Now()
	timeElapsed = time.Since(startTime)
	var packetcount uint64 = 0
	for timeElapsed.Milliseconds() < duration*2 {
		io.ReadFull(conn, packet)
		packetcount++
		timeElapsed = time.Since(startTime)
	}
	conn.Close()
}

func checkError(err error) {
	if err != nil {
		println(err)
		os.Exit(1)
	}
}
