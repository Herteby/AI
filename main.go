package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Primitive or Simple Types
type RunStatus string

const (
	StatusQueued         RunStatus = "queued"
	StatusInProgress     RunStatus = "in_progress"
	StatusRequiresAction RunStatus = "requires_action"
	StatusCancelling     RunStatus = "cancelling"
	StatusCancelled      RunStatus = "cancelled"
	StatusFailed         RunStatus = "failed"
	StatusCompleted      RunStatus = "completed"
	StatusExpired        RunStatus = "expired"
)

type Order string

const (
	Asc  Order = "asc"
	Desc Order = "desc"
)

// Basic Component Types
type Text struct {
	Value       string        `json:"value"`
	Annotations []interface{} `json:"annotations"` // Adjust based on actual structure
}

type Content struct {
	Type string `json:"type"`
	Text Text   `json:"text"`
}

type FunctionDefinition struct {
	Name            string `json:"name"`
	ArgumentsString string `json:"arguments"`
	Arguments       FunctionArguments
}
type FunctionArguments struct {
	Command string `json:"command"`
}
type FunctionObject struct {
	Description string                 `json:"description"`
	Name        string                 `json:"name"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Tool struct {
	Type     string         `json:"type"`
	Function FunctionObject `json:"function"`
}

type RunToolCallObject struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type ToolOutput struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
}

type SubmitToolOutputs struct {
	ToolCalls []RunToolCallObject `json:"tool_calls"`
}

type RequiredAction struct {
	Type              string            `json:"type"`
	SubmitToolOutputs SubmitToolOutputs `json:"submit_tool_outputs"`
}

// Core Entity Types
type Assistant struct {
	ID           string                 `json:"id"`
	CreatedAt    int64                  `json:"created_at"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Model        string                 `json:"model"`
	Instructions string                 `json:"instructions"`
	Tools        []string               `json:"tools"`
	FileIDs      []string               `json:"file_ids"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type AiThread struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"created_at"`
}

type Message struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	CreatedAt   int64                  `json:"created_at"`
	ThreadID    string                 `json:"thread_id"`
	Role        Role                   `json:"role"`
	Content     []Content              `json:"content"`
	FileIDs     []string               `json:"file_ids"`
	AssistantID *string                `json:"assistant_id"`
	RunID       *string                `json:"run_id"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type MessageObject struct {
	ID        string    `json:"id"`
	CreatedAt int64     `json:"created_at"`
	ThreadID  string    `json:"thread_id"`
	Role      Role      `json:"role"`
	Content   []Content `json:"content"`
}
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Run struct {
	ID             string                 `json:"id"`
	CreatedAt      int64                  `json:"created_at"`
	AssistantID    string                 `json:"assistant_id"`
	ThreadID       string                 `json:"thread_id"`
	Status         string                 `json:"status"`
	RequiredAction RequiredAction         `json:"required_action"`
	StartedAt      int64                  `json:"started_at"`
	ExpiresAt      *int64                 `json:"expires_at"`
	CancelledAt    *int64                 `json:"cancelled_at"`
	FailedAt       *int64                 `json:"failed_at"`
	CompletedAt    *int64                 `json:"completed_at"`
	LastError      *string                `json:"last_error"`
	Model          string                 `json:"model"`
	Instructions   *string                `json:"instructions"`
	Tools          []Tool                 `json:"tools"`
	FileIDs        []string               `json:"file_ids"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type RunStep struct {
	ID          string                 `json:"id"`
	CreatedAt   int64                  `json:"created_at"`
	RunID       string                 `json:"run_id"`
	AssistantID string                 `json:"assistant_id"`
	ThreadID    string                 `json:"thread_id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	CancelledAt *int64                 `json:"cancelled_at"`
	CompletedAt *int64                 `json:"completed_at"`
	ExpiredAt   *int64                 `json:"expired_at"`
	FailedAt    *int64                 `json:"failed_at"`
	LastError   *string                `json:"last_error"`
	StepDetails map[string]interface{} `json:"step_details"`
}

// Response Types
type ListAssistantsResponse struct {
	Data    []Assistant `json:"data"`
	FirstID string      `json:"first_id"`
	LastID  string      `json:"last_id"`
	HasMore bool        `json:"has_more"`
}

type ListMessagesResponse struct {
	Object  string    `json:"object"`
	Data    []Message `json:"data"`
	FirstID string    `json:"first_id"`
	LastID  string    `json:"last_id"`
	HasMore bool      `json:"has_more"`
}

type ListRunsResponse struct {
	Data    []Run  `json:"data"`
	FirstID string `json:"first_id"`
	LastID  string `json:"last_id"`
	HasMore bool   `json:"has_more"`
}

type ListRunStepsResponse struct {
	Data    []RunStep `json:"data"`
	FirstID string    `json:"first_id"`
	LastID  string    `json:"last_id"`
	HasMore bool      `json:"has_more"`
}

// Container or Utility Types
type Store struct {
	Thread AiThread `json:"thread"`
}

var debugMode bool

func main() {
	app := &cli.App{
		Name:  "AI Assistant CLI",
		Usage: "Interact with an AI assistant through the command line",
		Action: func(c *cli.Context) error {
			return startChat()
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Enable debug mode",
				Destination: &debugMode,
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

const assistantId = "asst_zE0QYAfDkGC9igpaDd0GaSx6"

func startChat() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Chat with AI Assistant. Type 'exit' to end the chat.")

	// Step 1: Get the thread
	thread, err := getThread()
	if err != nil {
		log.Fatal(err)
	}

	for {
		// Step 2: List runs and handle the first run
		runs, err := listRuns(thread.ID, PaginationParams{Limit: 1})
		if err != nil {
			log.Fatal(err)
		}
		var promptUser bool = false
		// If there are runs, handle the first run
		if len(runs.Data) > 0 {
			firstRun := runs.Data[0]
			status := RunStatus(firstRun.Status)

			switch status {
			case StatusQueued, StatusInProgress, StatusCancelling:
				// Poll every second until status changes
				for status == StatusQueued || status == StatusInProgress || status == StatusCancelling {
					fmt.Printf("Polling: %+v\n", status)
					time.Sleep(1 * time.Second)
					updatedRun, err := getRun(thread.ID, firstRun.ID)
					if err != nil {
						log.Fatal(err)
					}
					firstRun = *updatedRun
					status = RunStatus(firstRun.Status)
				}

			case StatusRequiresAction:
				var toolOutputs []ToolOutput
				for _, toolCall := range firstRun.RequiredAction.SubmitToolOutputs.ToolCalls {
					if toolCall.Function.Name == "terminal" {
						command := toolCall.Function.Arguments.Command

						// Execute the command
						cmdOutput, err := executeCommand(command)
						if err != nil {
							log.Fatalf("Failed to execute command: %s, error: %v", command, err)
						}

						// Append the output to toolOutputs
						toolOutputs = append(toolOutputs, ToolOutput{
							ToolCallID: toolCall.ID,
							Output:     cmdOutput,
						})
					}
				}

				// Submit tool outputs
				_, err = submitToolOutputsToRun(thread.ID, firstRun.ID, toolOutputs)
				if err != nil {
					log.Fatal(err)
				}
			case StatusCompleted:
				promptUser = true
			}

			// Step 3: Fetch and print messages (using CreatedAt as a cutoff)
			messages, err := listMessages(thread.ID, PaginationParams{
				Limit: 20,
				Order: Desc,
			})
			if err != nil {
				log.Fatal(err)
			}

			for _, message := range reverse(messages.Data) {
				switch message.Role {
				case RoleSystem:
					color.White("System: %s\n", message.Content[0].Text.Value)
				case RoleAssistant:
					color.Cyan("Assistant: %s\n", message.Content[0].Text.Value)
				case RoleUser:
					color.Yellow("You: %s\n", message.Content[0].Text.Value)

				}
			}
		} else {
			promptUser = true
		}

		// Step 4: Prompt user for message, send it to the thread, start a new run
		if promptUser {
			fmt.Print("You: ")
			userInput, _ := reader.ReadString('\n')
			userInput = strings.TrimSpace(userInput)

			if userInput == "exit" {
				break
			}

			_, err = createMessage(thread.ID, "user", userInput)
			if err != nil {
				log.Fatal(err)
			}

			_, err = createRun(thread.ID, assistantId)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	return nil
}

func executeCommand(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	outputBytes, err := cmd.CombinedOutput()
	return string(outputBytes), err
}

func getThread() (AiThread, error) {
	var store Store
	filename := "store.json"

	// Attempt to read and unmarshal the existing thread from store.json
	fileData, err := ioutil.ReadFile(filename)
	if err == nil {
		if json.Unmarshal(fileData, &store) == nil {
			return store.Thread, nil
		}
	}

	// Handle file not found or unmarshal error by creating a new thread
	if os.IsNotExist(err) || err != nil {
		thread, err := createThread()
		if err != nil {
			return AiThread{}, err
		}

		// Marshal the new thread into JSON and save it to store.json
		updatedData, err := json.Marshal(Store{Thread: thread})
		if err != nil {
			return AiThread{}, err
		}
		if err = ioutil.WriteFile(filename, updatedData, 0644); err != nil {
			return AiThread{}, err
		}
		return thread, nil
	}

	return AiThread{}, err
}

func runCommand(command string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
func reverse[T any](s []T) []T {
	result := make([]T, len(s))
	copy(result, s)

	for i := 0; i < len(result)/2; i++ {
		result[i], result[len(result)-1-i] = result[len(result)-1-i], result[i]
	}

	return result
}
