package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type MoonBix struct {
	baseHeaders map[string]string
	MIN_WIN     int
	MAX_WIN     int
}

func NewMoonBix() *MoonBix {
	return &MoonBix{
		baseHeaders: map[string]string{
			"Accept":              "application/dp, text/plain, */*",
			"User-Agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			"Content-Type":        "application/dp",
			"Origin":              "https://www.binance.com/game/tg/moon-bix",
			"X-Requested-With":    "org.telegram.messenger",
			"Referer":             "https://www.binance.com/game/tg/moon-bix",
			"Accept-Encoding":     "gzip, deflate",
			"Accept-Language":     "en,en-US;q=0.9",
		},
	}
}

func (mb *MoonBix) log(message string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] %s\n", now, message)
}

func (mb *MoonBix) httpRequest(method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return client.Do(req)
}

func (mb *MoonBix) renewAccessToken(tgData string) (string, error) {
	url := "https://www.binance.com/bapi/growth/v1/friendly/growth-paas/mini-app-activity/third-party/user/user-info"
	headers := mb.baseHeaders
	data := map[string]string{"query": tgData}
	body, _ := json.Marshal(data)

	res, err := mb.httpRequest("GET", url, headers, body)
	if err != nil {
		return "", err
	}


	fmt.Println(res)

	defer res.Body.Close()

	var result map[string]interface{}

	json.NewDecoder(res.Body).Decode(&result)

	token, ok := result["token"].(map[string]interface{})
	if !ok {
		mb.log("token not found, please get a new one")
		return "", nil
	}

	accessToken, ok := token["access"].(string)
	if !ok {
		mb.log("token was successfully loaded")
		return "", nil
	}

	return accessToken, nil
}

func (mb *MoonBix) solve(task map[string]interface{}, accessToken string) {
	headers := mb.baseHeaders
	headers["Authorization"] = "Bearer " + accessToken

	taskID := task["id"].(string)
	taskStatus := task["status"].(string)

	startTaskURL := fmt.Sprintf("https://www.binance.com/bapi/growth/v1/friendly/growth-paas/mini-app-activity/third-party/task/list/%s", taskID)
	claimTaskURL := startTaskURL

	if taskStatus == "job done" {
		mb.log(fmt.Sprintf("already claimed task id %s", taskID))
		return
	}

	if taskStatus == "READY_FOR_CLAIM" {
		mb.httpRequest("POST", claimTaskURL, headers, nil)
		mb.log(fmt.Sprintf("successfully completed task id %s", taskID))
		return
	}

	mb.httpRequest("POST", startTaskURL, headers, nil)
	time.Sleep(5 * time.Second)
	mb.httpRequest("POST", claimTaskURL, headers, nil)
}

func (mb *MoonBix) solveTask(accessToken string) {
	urlTasks := "https://www.binance.com/bapi/growth/v1/friendly/growth-paas/mini-app-activity/third-party/task/list"
	headers := mb.baseHeaders
	headers["Authorization"] = "Bearer " + accessToken

	res, err := mb.httpRequest("GET", urlTasks, headers, nil)
	if err != nil {
		mb.log("failed to fetch tasks!")
		return
	}
	defer res.Body.Close()

	var tasksResponse map[string]interface{}
	json.NewDecoder(res.Body).Decode(&tasksResponse)

	tasks, ok := tasksResponse["tasks"].([]interface{})
	if !ok {
		mb.log("failed to retrieve tasks!")
		return
	}

	for _, t := range tasks {
		if taskMap, ok := t.(map[string]interface{}); ok {
			mb.solve(taskMap, accessToken)
		}
	}
}

func (mb *MoonBix) loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		mb.log("failed to open config.json")
		return
	}
	defer file.Close()

	var config struct {
		MIN_WIN int `json:"game_point.low"`
		MAX_WIN int `json:"game_point.high"`
	}

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		mb.log("failed to decode config.json")
		return
	}

	mb.MIN_WIN = config.MIN_WIN
	mb.MAX_WIN = config.MAX_WIN

	if mb.MIN_WIN > mb.MAX_WIN {
		mb.log("high value must be higher than lower value")
		os.Exit(1)
	}
}

func main() {
	mb := NewMoonBix()
	mb.loadConfig()
	mb.log("MoonBix started")

	// Example of how to renew access token and solve tasks
	accessToken, err := mb.renewAccessToken("query_id=xxxxx")
	if err != nil {
		log.Fatal(err)
	}

	mb.solveTask(accessToken)
}
