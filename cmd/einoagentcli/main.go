package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"meetingagent/cmd/einoagent/agent"
	"meetingagent/einoagent"
	"meetingagent/pkg/env"
	"meetingagent/pkg/mem"
	"os"
	"strconv"
	"strings"

	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

var id = flag.String("id", "", "conversation id")

var memory = mem.GetDefaultMemory()

var cbHandler callbacks.Handler

func main() {
	flag.Parse()

	// 开启 Eino 的可视化调试能力
	err := devops.Init(context.Background())
	if err != nil {
		log.Printf("[eino dev] init failed: %v", err)
		return
	}

	if *id == "" {
		*id = strconv.Itoa(rand.Intn(1000000))
	}

	ctx := context.Background()

	err = Init()
	if err != nil {
		log.Printf("[eino agent] init failed: %v", err)
		return
	}

	// start interactive dialogue
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("🧑‍ : ")

		// Read the input from the user
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v", err)
			return
		}

		input = strings.TrimSpace(input)
		if input == "" || input == "exit" || input == "quit" {
			return
		}

		// call RunAgent with the input
		sr, err := agent.RunAgent(ctx, *id, input)
		if err != nil {
			fmt.Printf("Error from RunAgent: %v", err)
			continue
		}

		fmt.Print("🤖 : ")
		for {
			msg, err := sr.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Printf("Error receiving message: %v", err)
				break
			}
			fmt.Print(msg.Content)
		}
		fmt.Println()
		fmt.Println()
	}
}

func Init() error {
	env.MustHasEnvs("ARK_CHAT_MODEL", "ARK_EMBEDDING_MODEL", "ARK_API_KEY")

	os.MkdirAll("log", 0755)
	var f *os.File
	f, err := os.OpenFile("log/eino.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	cbConfig := &LogCallbackConfig{
		Detail: true,
		Writer: f,
	}

	if os.Getenv("DEBUG") == "true" {
		cbConfig.Debug = true
	}
	cbHandler = LogCallback(cbConfig)

	callbackHandlers := make([]callbacks.Handler, 0)
	if os.Getenv("APMPLUS_APP_KEY") != "" {
		region := os.Getenv("APMPLUS_REGION")
		if region == "" {
			region = "cn-beijing"
		}
		fmt.Println("[eino agent] INFO: use apmplus as callback")
		cbh, _, err := apmplus.NewApmplusHandler(&apmplus.Config{
			Host:        fmt.Sprintf("apmplus-%s.volces.com:4317", region),
			AppKey:      os.Getenv("APMPLUS_APP_KEY"),
			ServiceName: "eino-meeting-agent",
			Release:     "release/v0.0.1",
		})
		if err != nil {
			log.Fatal(err)
		}
		callbackHandlers = append(callbackHandlers, cbh)
	}

	if len(callbackHandlers) > 0 {
		callbacks.InitCallbackHandlers(callbackHandlers)
	}

	return nil
}

func RunAgent(ctx context.Context, id string, msg string) (*schema.StreamReader[*schema.Message], error) {
	runner, err := einoagent.BuildEinoAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent graph: %v", err)
	}

	conversation := memory.GetConversation(id, true)

	userMessage := &einoagent.UserMessage{
		ID:      id,
		Query:   msg,
		History: conversation.GetMessages(),
	}

	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(cbHandler))
	if err != nil {
		return nil, fmt.Errorf("failed to stream: %w", err)
	}

	srs := sr.Copy(2)

	go func() {
		// for save to memory
		fullMsgs := make([]*schema.Message, 0)

		defer func() {
			srs[1].Close()

			conversation.Append(schema.UserMessage(msg))

			fullMsg, err := schema.ConcatMessages(fullMsgs)
			if err != nil {
				fmt.Println("error concatenating messages: ", err.Error())
			}
			conversation.Append(fullMsg)
		}()

	outer:
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context done", ctx.Err())
				return
			default:
				chunk, err := srs[1].Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break outer
					}
				}
				fullMsgs = append(fullMsgs, chunk)
			}
		}
	}()

	return srs[0], nil
}

type LogCallbackConfig struct {
	Detail bool
	Debug  bool
	Writer io.Writer
}

func LogCallback(config *LogCallbackConfig) callbacks.Handler {
	if config == nil {
		config = &LogCallbackConfig{
			Detail: true,
			Writer: os.Stdout,
		}
	}
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	builder := callbacks.NewHandlerBuilder()
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		fmt.Fprintf(config.Writer, "[view]: start [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		if config.Detail {
			var b []byte
			if config.Debug {
				b, _ = json.MarshalIndent(input, "", "  ")
			} else {
				b, _ = json.Marshal(input)
			}
			fmt.Fprintf(config.Writer, "%s\n", string(b))
		}
		return ctx
	})
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		fmt.Fprintf(config.Writer, "[view]: end [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		return ctx
	})
	return builder.Build()
}
