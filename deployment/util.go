package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func readFileOrPanic(path string, ctx *pulumi.Context) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}
	return string(data)
}

func getAccessToken(url string, token string) string {
	// Querying the Consul KV store to fetch the access token
	client := &http.Client{}

	// Retry till Consul is ready
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", url+"nomad_user_token", nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			time.Sleep(time.Second * 20)
			log.Println("Retrying...")
		} else {
			defer resp.Body.Close()
			// Read the response body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			// Extract the value from the response body
			var response []struct {
				Value string `json:"Value"`
			}
			err = json.Unmarshal(body, &response)
			if err != nil {
				log.Fatal(err)
			}

			if len(response) > 0 {
				// Resp is base64 encoded, TODO figure out better way
				value64 := response[0].Value
				value, err := base64.StdEncoding.DecodeString(value64)
				if err != nil {
					log.Fatal(err)
				}

				return string(value)
			} else {
				log.Fatal("Value not found in the response")
				return ""
			}

		}
	}

	return "Timeout error"

}

func setAccountKey(url string, key64 string, token string) []byte {
	// Querying the Consul KV store to fetch the access token
	client := &http.Client{}
	key, err := base64.StdEncoding.DecodeString(key64)
	if err != nil {
		log.Fatal(err)
	}

	// Retry till Consul is ready
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest(http.MethodPut, url+"service_account", bytes.NewBuffer([]byte(key)))
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			time.Sleep(time.Second * 15)
			log.Println("Retrying...")
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			return body
		}
	}

	return []byte("Timeout error")
}

func injectToken(token string, toBeReplaced string, script string, amount int) string {

	return strings.Replace(script, toBeReplaced, token, amount)
}
func createToken() string {
	return uuid.NewString()
}
