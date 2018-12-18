package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/jdxcode/netrc"
	"github.com/justjanne/powerline-go/powerline"
	"github.com/zorkian/go-datadog-api"
)

const (
	Green  = 35
	Yellow = 230
	Grey   = 250
	Red    = 161
)

type monitorCounter struct {
	Ok     int
	Warn   int
	Alert  int
	NoData int
}

const tempFile = "/tmp/powerline-go-dd.json"

func appendMonitor(segments []*powerline.Segment, title string, color uint8, count int) []*powerline.Segment {
	if count > 0 {
		return append(segments, &powerline.Segment{
			Content:    fmt.Sprintf("%d", count),
			Background: color,
		})
	}
	return segments
}

func main() {
	if os.Getenv("POWERLINE_GO_DATADOG_REFRESH") == "true" {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		n, err := netrc.Parse(filepath.Join(usr.HomeDir, ".netrc"))
		if err != nil {
			panic(err)
		}
		machine := n.Machine("api.datadoghq.com")

		client := datadog.NewClient(machine.Get("password"), machine.Get("login"))

		monitors, err := client.GetMonitors()
		if err != nil {
			log.Fatalf("fatal: %s\n", err)
		}

		monitorCounter := monitorCounter{
			Ok:     0,
			Warn:   0,
			Alert:  0,
			NoData: 0,
		}

		for _, monitor := range monitors {
			switch monitor.GetOverallState() {
			case "OK":
				monitorCounter.Ok++
			case "No Data":
				monitorCounter.NoData++
			case "Warn":
				monitorCounter.Warn++
			case "Alert":
				monitorCounter.Alert++
			}
		}

		segments := []*powerline.Segment{
			&powerline.Segment{
				Content: "DD",
			},
		}
		segments = appendMonitor(segments, "Ok", Green, monitorCounter.Ok)
		segments = appendMonitor(segments, "Warn", Yellow, monitorCounter.Warn)
		segments = appendMonitor(segments, "Alert", Red, monitorCounter.Alert)
		segments = appendMonitor(segments, "No Data", Grey, monitorCounter.NoData)

		fd, err := os.Create(tempFile + ".new")
		if err != nil {
			panic(err)
		}

		e := json.NewEncoder(fd)
		err = e.Encode(segments)
		if err != nil {
			panic(err)
		}
		err = os.Rename(tempFile+".new", tempFile)
		if err != nil {
			panic(err)
		}
	} else {
		s, err := os.Stat(tempFile)
		if err == nil {
			fd, err := os.Open(tempFile)
			if err != nil {
				panic(err)
			}
			_, err = io.Copy(os.Stdout, fd)
			if err != nil {
				panic(err)
			}
		}
		if (err != nil && os.IsNotExist(err)) || s.ModTime().Before(time.Now().Add(-30*time.Second)) {
			cmd := exec.Command(os.Args[0])
			cmd.Env = []string{"POWERLINE_GO_DATADOG_REFRESH=true"}
			err := cmd.Start()
			if err != nil {
				panic(err)
			}
			err = cmd.Process.Release()
			if err != nil {
				panic(err)
			}
		}
	}
}
