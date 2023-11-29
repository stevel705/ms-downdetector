package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron/v3"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type ServiceStatus struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	Code   int    `json:"code,omitempty"`
	Error  string `json:"error,omitempty"`
}

var vpsServers map[string][]string

var failedCounts = make(map[string]int)
var mtx sync.Mutex

func init() {
	data, err := os.ReadFile("./vps_servers.json")
	if err != nil {
		log.Fatalf("Error reading VPS servers file: %v", err)
	}
	err = json.Unmarshal(data, &vpsServers)
	if err != nil {
		log.Fatalf("Error parsing VPS servers file: %v", err)
	}
}

func checkServiceStatus(url string) ServiceStatus {

	// Create a custom HTTP client with a timeout
	client := http.Client{
		Timeout: 1 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		mtx.Lock()
		failedCounts[url]++
		mtx.Unlock()
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return ServiceStatus{URL: url, Status: "DOWN", Error: "Request timed out"}
		}
		return ServiceStatus{URL: url, Status: "DOWN", Error: "Unable to connect"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		mtx.Lock()
		failedCounts[url] = 0
		mtx.Unlock()
		return ServiceStatus{URL: url, Status: "UP"}
	}
	mtx.Lock()
	failedCounts[url]++
	mtx.Unlock()
	return ServiceStatus{URL: url, Status: "DOWN", Code: resp.StatusCode}
}

var telegramToken = os.Getenv("TELEGRAM_TOKEN")
var chatIDStr = os.Getenv("CHAT_ID")
var chatID, _ = strconv.ParseInt(chatIDStr, 10, 64) // Convert string to int64
// if err != nil {
// log.Fatalf("Failed to convert CHAT_ID to integer: %v", err)
// }

func main() {
	r := gin.Default()

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Panic(err)
	}

	c := cron.New()
	c.AddFunc("@every 1m", func() {
		for vpsName, services := range vpsServers {
			for _, url := range services {
				status := checkServiceStatus(url)
				if failedCounts[url] >= 3 {
					msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Service %s is down! From %s", url, vpsName))
					bot.Send(msg)
					mtx.Lock()
					failedCounts[url] = 0
					mtx.Unlock()
				}
				fmt.Println(status)
			}
		}
	})
	c.Start()

	r.GET("/check", func(c *gin.Context) {
		vpsParam := c.QueryArray("vps")
		results := make(map[string][]ServiceStatus)
		for vpsName, services := range vpsServers {
			if len(vpsParam) == 0 || contains(vpsParam, vpsName) {
				for _, url := range services {
					results[vpsName] = append(results[vpsName], checkServiceStatus(url))
				}
			}
		}
		c.JSON(200, results)
	})

	r.Run()
}

func contains(s []string, searchterm string) bool {
	for _, v := range s {
		if v == searchterm {
			return true
		}
	}
	return false
}
