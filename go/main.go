package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron/v3"
)

type ServiceStatus struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	Code   int    `json:"code,omitempty"`
	Error  string `json:"error,omitempty"`
}

var vpsServers = map[string][]string{
	"vps_1_test": {"https://project.web-ar.studio/health", "http://109.248.175.153:5050/api/health"},
	"vps_2_prod": {"https://web-ar.studio/en/", "http://109.248.175.87:5050/api/health", "http://109.248.175.87:8080/authService/helth"},
	"vps_3_test": {"http://207.154.216.146:3001/authService/helth", "http://207.154.216.146:8081/__health"},
	"vps_4_prod": {"http://109.248.175.228:5000/api/v1/image-tracking/"},
	"vps_5_test": {"https://test.web-ar.studio/en/"},
	"vps_6_test": {"http://139.59.140.141:5001/api/v1/image-tracking/"},

	// Add more VPS and services as needed
}

var failedCounts = make(map[string]int)
var mtx sync.Mutex

func checkServiceStatus(url string) ServiceStatus {
	resp, err := http.Get(url)
	if err != nil {
		mtx.Lock()
		failedCounts[url]++
		mtx.Unlock()
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
		for _, services := range vpsServers {
			for _, url := range services {
				status := checkServiceStatus(url)
				if failedCounts[url] >= 3 {
					msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Service %s is down! From %s", url, services))
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
