package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/tidwall/pretty"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type HTTPMethod string

// HTTP method constants
const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
)

func Api(method HTTPMethod, url string, requestBody interface{}, responseStruct interface{}) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OpenAI API key is not set")
	}

	var reqBody io.Reader
	var requestBodyDebug []byte
	var err error

	if requestBody != nil {
		requestBodyDebug, err = json.Marshal(requestBody)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(requestBodyDebug)
	}

	req, err := http.NewRequest(string(method), url, reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v1")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Debugging: Print URL, Request Body, and Response Body
	if debugMode {
		color.Cyan("URL: %s", url)
		if requestBody != nil {
			colorizedRequestBody := pretty.Color(requestBodyDebug, nil)
			fmt.Println(string(colorizedRequestBody))
		}

		bodyBytes, _ := ioutil.ReadAll(response.Body)
		colorizedResponseBody := pretty.Color(bodyBytes, nil)
		fmt.Println(string(colorizedResponseBody))
		response.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset response body for decoding
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("API call failed with status: %d", response.StatusCode)
	}

	return json.NewDecoder(response.Body).Decode(responseStruct)
}

func createThread() (AiThread, error) {

	var thread AiThread
	err := Api(POST, "https://api.openai.com/v1/threads", nil, &thread)
	if err != nil {
		return AiThread{}, err
	}
	if thread.ID == "" {
		log.Fatal("empty thread ID")
	}

	return thread, nil
}
func createMessage(threadID, role, content string) (*MessageObject, error) {
	requestBody := map[string]string{"role": role, "content": content}

	var messageObj MessageObject
	err := Api(POST, "https://api.openai.com/v1/threads/"+threadID+"/messages", requestBody, &messageObj)
	if err != nil {
		return nil, err
	}

	return &messageObj, nil
}

func getRun(threadID string, runID string) (*Run, error) {
	url := "https://api.openai.com/v1/threads/" + threadID + "/runs/" + runID

	var run Run
	err := Api(GET, url, nil, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func createRun(threadID string, assistantID string) (*Run, error) {
	url := "https://api.openai.com/v1/threads/" + threadID + "/runs"
	requestBody := map[string]string{"assistant_id": assistantID}

	var run Run
	err := Api(POST, url, requestBody, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}
func submitToolOutputsToRun(threadID string, runID string, toolOutputs []ToolOutput) (*Run, error) {
	url := "https://api.openai.com/v1/threads/" + threadID + "/runs/" + runID + "/submit_tool_outputs"

	// Directly constructing the request body using an anonymous struct
	requestBody := struct {
		ToolOutputs []ToolOutput `json:"tool_outputs"`
	}{
		ToolOutputs: toolOutputs,
	}

	var run Run
	err := Api(POST, url, requestBody, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func cancelRun(threadID string, runID string) (*Run, error) {
	url := "https://api.openai.com/v1/threads/" + threadID + "/runs/" + runID + "/cancel"

	var run Run
	err := Api(POST, url, nil, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

type PaginationParams struct {
	Limit  int
	Order  Order
	After  string
	Before string
}

// More descriptive function name
func (p PaginationParams) Encode() string {
	values := url.Values{}

	if p.Limit != 0 {
		values.Add("limit", strconv.Itoa(p.Limit))
	}
	if p.Order != "" {
		values.Add("order", string(p.Order))
	}
	if p.After != "" {
		values.Add("after", p.After)
	}
	if p.Before != "" {
		values.Add("before", p.Before)
	}

	return values.Encode()
}

// listRuns function with streamlined URL concatenation
func listRuns(threadID string, pagination PaginationParams) (*ListRunsResponse, error) {
	var responseStruct ListRunsResponse
	err := Api(GET, "https://api.openai.com/v1/threads/"+threadID+"/runs?"+pagination.Encode(), nil, &responseStruct)
	if err != nil {
		return nil, fmt.Errorf("listRuns API call failed: %w", err)
	}

	return &responseStruct, nil
}

func listRunSteps(threadID string, runID string, pagination PaginationParams) (*ListRunStepsResponse, error) {
	var responseStruct ListRunStepsResponse
	err := Api(GET, "https://api.openai.com/v1/threads/"+threadID+"/runs/"+runID+"/steps?"+pagination.Encode(), nil, &responseStruct)
	if err != nil {
		return nil, err
	}

	return &responseStruct, nil
}

func listMessages(threadID string, pagination PaginationParams) (*ListMessagesResponse, error) {
	var responseStruct ListMessagesResponse
	err := Api(GET, "https://api.openai.com/v1/threads/"+threadID+"/messages?"+pagination.Encode(), nil, &responseStruct)
	if err != nil {
		return nil, err
	}

	return &responseStruct, nil
}
