package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tatsushid/go-fastping"
	"github.com/urfave/cli"
)

func catch(e error) {
	if e != nil {
		log.Fatalf("Error: %v", e)
	}
}

func pingger(host string, size int) {

	type response struct {
		addr *net.IPAddr
		rtt  time.Duration
	}

	p := fastping.NewPinger()

	ra, err := net.ResolveIPAddr("ip", host)
	if err == nil {
		results := make(map[string]*response)
		results[ra.String()] = nil
		p.AddIPAddr(ra)

		onRecv, onIdle := make(chan *response), make(chan bool)

		p.OnRecv = func(addr *net.IPAddr, t time.Duration) {
			onRecv <- &response{addr: addr, rtt: t}
		}
		p.OnIdle = func() {
			onIdle <- true
		}

		p.MaxRTT = time.Second
		p.RunLoop()

	loop:
		for {
			select {
			case res := <-onRecv:
				if _, ok := results[res.addr.String()]; ok {
					results[res.addr.String()] = res
				}
			case <-onIdle:
				for addr, r := range results {
					if r == nil {
						display(host, addr, "NG", 0*time.Second)
					} else {
						display(host, addr, "OK", r.rtt)
					}
					results[addr] = nil
				}
				break loop
			}
		}
	} else {
		display(host, "Unresolved", "NG", 0*time.Second)
	}
}

func display(host string, addr string, pod string, rtt time.Duration) {
	if rtt == 0 {
		fmt.Printf("[%s] HOST: %s IP: %s RTT: --\n", pod, host, addr)
	} else {
		fmt.Printf("[%s] HOST: %s IP: %s RTT: %v\n", pod, host, addr, rtt)
	}

}

func worker(iplist []string, cnt int, slp time.Duration, opt map[string]int) {
	var wg sync.WaitGroup
	for i := 0; ; {
		i++
		fmt.Printf("%s", strings.Repeat("-", 20))
		fmt.Printf(" Round: %d ", i)
		fmt.Printf("%s \n", strings.Repeat("-", 20))
		if i == cnt {
			break
		}
		for _, ip := range iplist {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				pingger(ip, opt["size"])
			}(ip)
		}
		wg.Wait()
		time.Sleep(slp)
	}
}

func readlines(filename string) []string {
	var iplist []string
	fd, err := os.Open(filename)
	catch(err)
	defer fd.Close()

	sc := bufio.NewScanner(fd)
	for sc.Scan() {
		line := sc.Text()
		if line[:1] == "#" {
			continue
		}
		iplist = append(iplist, line)
	}
	return iplist
}

func main() {

	app := cli.NewApp()
	app.Name = "Pong or Dead"
	app.Usage = "ping to listed target"
	app.Version = "0.0.1"

	type pingOpt struct {
		size int
		udp  bool
	}

	var (
		count  int
		sleep  int
		size   int
		udp    bool
		option map[string]int = map[string]int{
			"size": 0,
		}
	)

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "Count, c",
			Value:       0,
			Usage:       "Count for ping routine",
			Destination: &count,
		},
		cli.IntFlag{
			Name:        "Sleep, s",
			Value:       1,
			Usage:       "Sleep time for ping routine (s)",
			Destination: &sleep,
		},
		cli.IntFlag{
			Name:        "Size, S",
			Value:       64,
			Usage:       "Ping size (Byte)",
			Destination: &size,
		},
		cli.BoolFlag{
			Name:        "UDP, u",
			Usage:       "use UDP ping",
			Destination: &udp,
		},
	}

	app.Action = func(c *cli.Context) error {
		if c.NArg() == 0 {
			fmt.Printf("Usage: %s --help\n", os.Args[0])
			os.Exit(1)
		}

		option["size"] = size

		worker(readlines(c.Args().Get(0)), count, time.Duration(sleep)*time.Second, option)
		return nil
	}
	err := app.Run(os.Args)
	catch(err)
}
