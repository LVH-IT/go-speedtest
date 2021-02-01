package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var duration int64
var packetSize int
var ppiDownload []uint64
var ppiUpload []uint64

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
	duration = 10
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
	var packetsPerInterval []uint64
	for i := 0; i < 2; i++ {
		var packetcount uint64 = 0

		testDone := false
		startTime := time.Now()
		go func() {
			timerStartTimeSince := time.Since(startTime).Seconds()
			for !testDone {
				for len(packetsPerInterval) == 0 {
					time.Sleep(1e6)
				}
				if i == 0 {
					fmt.Printf("\rTesting download Speed (%ds/"+fmt.Sprint(duration/1e3)+"s) | %.2fMbit/s", int(timerStartTimeSince), calcRate(packetsPerInterval[len(packetsPerInterval)-1], time.Millisecond*100))
				} else if i == 1 {
					fmt.Printf("\rTesting upload Speed (%ds/"+fmt.Sprint(duration/1e3)+"s) | %.2fMbit/s", int(timerStartTimeSince), calcRate(packetsPerInterval[len(packetsPerInterval)-1], time.Millisecond*100))
				}
				time.Sleep(10e6)
				timerStartTimeSince = time.Since(startTime).Seconds()
			}
		}()

		go func() {
			packetsPerInterval = append(packetsPerInterval, 0)
			for i := 0; i < int((duration/1e3))*10; i++ {
				time.Sleep(100 * time.Millisecond)
				packetsPerInterval = append(packetsPerInterval, packetcount)
				packetcount = 0
			}
			testDone = true
		}()

		var err error
		for !testDone {
			if i == 0 {
				_, err = io.ReadFull(conn, packet)
			} else if i == 1 {
				_, err = conn.Write(packet)
			}
			if err == nil {
				packetcount++

			}
		}

		var fullPacketCount uint64
		if i == 0 {
			ppiDownload = packetsPerInterval
		} else if i == 1 {
			ppiUpload = packetsPerInterval
		}
		for _, b := range packetsPerInterval {
			fullPacketCount += b
		}
		packetsPerInterval = make([]uint64, 0)
		dura, err := time.ParseDuration(fmt.Sprint(duration) + "ms")
		checkError(err)
		if i == 0 {
			println()
			println("Average Download Speed: ", fmt.Sprintf("%.2f", calcRate(fullPacketCount, dura)), " Mbit/s")
		} else if i == 1 {
			println()
			println("Average Upload Speed: ", fmt.Sprintf("%.2f", calcRate(fullPacketCount, dura)), " Mbit/s")
		}
	}

	chartDataDL := ""
	chartLabelsDL := ""
	for a, b := range ppiDownload {
		chartDataDL += fmt.Sprintf("%.2f", calcRate(b, time.Millisecond*100)) + ","
		chartLabelsDL += "'" + fmt.Sprint(a*100) + "'" + ","
	}
	chartDataDL = strings.TrimSuffix(chartDataDL, ",")
	chartLabelsDL = strings.TrimSuffix(chartLabelsDL, ",")

	chartDataUL := ""
	chartLabelsUL := ""
	for a, b := range ppiUpload {
		chartDataUL += fmt.Sprintf("%.2f", calcRate(b, time.Millisecond*100)) + ","
		chartLabelsUL += "'" + fmt.Sprint(a*100) + "'" + ","
	}
	chartDataUL = strings.TrimSuffix(chartDataUL, ",")
	chartLabelsUL = strings.TrimSuffix(chartLabelsUL, ",")

	println("Generatin Report")
	htmlSampleBytes, err := ioutil.ReadFile("sample.html")
	checkError(err)
	html := strings.Replace(string(htmlSampleBytes), "LABELSGOHERE", chartLabelsDL, -1)
	html = strings.Replace(html, "DOWNLOADDATAGOESHERE", chartDataDL, -1)
	html = strings.Replace(html, "UPLOADDATAGOESHERE", chartDataUL, -1)
	htmlFile, err := os.Create("speedtest.html")
	checkError(err)
	_, err = io.WriteString(htmlFile, html)
	checkError(err)
	println("Done")
}

func calcRate(packetCount uint64, timeElapsed time.Duration) float64 {
	//+58 bytes for the non data parts of the tcp packets-> http://www.firewall.cx/networking-topics/protocols/tcp/138-tcp-options.html
	return (((float64(packetCount) / (float64(timeElapsed.Nanoseconds()) / 1e9)) / (1e6 / float64(packetSize+58))) * 8)
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
	log.Println("connected to: " + conn.RemoteAddr().String())
	packet := make([]byte, packetSize)
	startTime := time.Now()
	timeElapsed := time.Since(startTime)
	log.Println("Starting download test")
	for timeElapsed.Milliseconds() < duration {
		conn.Write(packet)
		timeElapsed = time.Since(startTime)
	}
	startTime = time.Now()
	timeElapsed = time.Since(startTime)
	log.Println("Starting upload test")
	for timeElapsed.Milliseconds() < duration {
		io.ReadFull(conn, packet)
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
