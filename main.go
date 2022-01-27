package main

import (
"bytes"
"flag"
"fmt"
"github.com/segmentio/ksuid"
"io/ioutil"
"math/rand"
"net/http"
"sync"
"time"
)

const (
	stageUrl = "https://api.segment.build/v1/track"
	prodUrl  = "https://api.segment.io/v1/track"
	euProdUrl = "https://tracking-api.euw1.segment.com"
)

var (
	environment = "production"
	eventType = "identify"
	debug       = false

	upperLetters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	lowerLetters = []rune("abcdefghijklmnopqrstuvwxyz")
	doneCount = 0
)

func main() {
	writeKeyFlag := flag.String("writeKey", "", "The write key of the source you are sending events to")
	numCreateUsersFlag := flag.Int("numUsers", 0, "The number of users to send")
	maxConcurrentFlag := flag.Int("maxConcurrent", 20, "The number of concurrent goroutines to make API calls")
	envFlag := flag.String("env", "production", "The environment to send events to [stage | production]")
	eventTypeFlag := flag.String("eventType", "identify", "The type of event to send to the tracking API [identify | track]")
	debugFlag := flag.Bool("debug", false, "Turn on debug messages")
	flag.Parse()

	writeKey := *writeKeyFlag
	numCreateUsers := *numCreateUsersFlag
	maxRoutines := *maxConcurrentFlag
	environment = *envFlag
	eventType = *eventTypeFlag
	debug = *debugFlag



	if writeKey == "" {
		fmt.Println("writeKey is required")
		return
	}

	if numCreateUsers < 1 {
		fmt.Println("numUsers is required and must be greater than 0")
		return
	}

	if maxRoutines < 1 || maxRoutines > 299 {
		fmt.Println("maxConcurrent should be greater than 0 but less than 300")
		return
	}

	if environment != "stage" && environment != "production" {
		fmt.Println("Environment must be set to stage or production")
		return
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	var wg sync.WaitGroup

	jobChan := make(chan struct{}, maxRoutines)
	resultChan := make(chan bool, maxRoutines)

	for i := 0; i < maxRoutines; i++ {
		wg.Add(1)
		go sender(client, writeKey, jobChan, resultChan, &wg)
	}

	fmt.Printf("Hi there! I'm going to attempt to send %d %s events to a workspace.\n", numCreateUsers, eventType)
	startTime := time.Now()
	fmt.Printf("Began sending %s events at: %s\n", eventType, startTime.Format(time.RFC3339))
	go func() {
		for i := 0; i < numCreateUsers; i++ {
			jobChan <- struct{}{}
		}
		close(jobChan)
	}()

	// Maybe add a way of calculating what rps events are being sent at

	successfulReqs := 0
	failedReqs := 0
	go func() {
		for result := range resultChan {
			if result {
				successfulReqs++
			} else {
				failedReqs++
			}
		}
	}()

	wg.Wait()
	time.Sleep(500000)

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	fmt.Printf("Completed sending events at: %s\n", endTime.Format(time.RFC3339))
	fmt.Printf("Number of successful calls: %d\n", successfulReqs)
	fmt.Printf("Number of failed calls: %d\n", failedReqs)
	fmt.Printf("Time duration: %s\n", duration)
}

func sender(client *http.Client, writeKey string, jobChan chan struct{}, resultChan chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for range jobChan {
		randomUserId := "use_" + ksuid.New().String()
		err := makeTrackRequest(client, writeKey, randomUserId)
		if err != nil {
			if debug {
				fmt.Printf("Failed to send track event: %s\n", err)
			}
			resultChan <- false
		} else {
			if debug {
				doneCount++
				fmt.Printf("Successfully created user %s, and count is: %v \n", randomUserId, doneCount)
			}
			resultChan <- true
		}

		time.Sleep(5 * time.Second)
	}
}

func makeTrackRequest(client *http.Client, writeKey, userId string) error {
	var url string
	if environment == "stage" {
		url = stageUrl
	} else {
		url = prodUrl
	}
	//Load Test #1
	var trackBody string
	if eventType == "identify" {
		trackBody = fmt.Sprintf(`{
         "userId": "%s",
         "type": "identify",
         "context": {},
         "integrations": {},
          "event": "Bought Joggers",
          "traits": {
             "Bought Joggers": true
          }
      }`, userId)
	} else {
		//mobileId := generateRandomMobileID()
		email := userId + "@segment-x.com"
		firstName, lastName := generateRandomName()
		crmId := generateRandomCrmId()
		trackBody = fmt.Sprintf(`{
		 "userId": "%s",
         "event": "Adwords Test",
         "properties": {
			"name": "some property",
    		"revenue": 14.99
 			 },
  		"context": {
			"traits" : {
                "name" : "%s, %s",
             	"email": "%s",
                "crmId": "%s"
			},
    		"ip": "24.5.68.47"
        }
      }`, userId, lastName, firstName, email,crmId)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(trackBody)))
	if err != nil {
		return fmt.Errorf("failed to create http request: %s", err)
	}

	req.SetBasicAuth(writeKey+":", "")
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("error during http request: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, body)
	}

	return nil
}

func generateRandomName() (string, string) {
	firstName := make([]rune, 8)
	lastName := make([]rune, 8)
	for i := 0; i < 8; i++ {
		if i == 0 {
			firstName[i] = upperLetters[rand.Intn(len(upperLetters))]
			lastName[i] = upperLetters[rand.Intn(len(upperLetters))]
		} else {
			firstName[i] = lowerLetters[rand.Intn(len(lowerLetters))]
			lastName[i] = lowerLetters[rand.Intn(len(lowerLetters))]
		}
	}
	return string(firstName), string(lastName)
}

func generateRandomMobileID() string {
	part1 := rand.Intn(89999) + 10000
	part2 := rand.Intn(89999) + 10000
	part3 := rand.Intn(89999) + 10000
	return fmt.Sprintf("%d-%d-%d", part1, part2, part3)
}

func generateRandomCrmId() string {
	part1 := rand.Intn(89999) + 10000
	return fmt.Sprintf("%d", part1)
}
