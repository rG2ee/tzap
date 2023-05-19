package tzap

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/tzapio/tzap/pkg/types"
	"github.com/tzapio/tzap/pkg/types/openai"
	"github.com/tzapio/tzap/pkg/util"
)

func GetThread(t *Tzap) []types.Message {
	messages := getThread(t)
	if t.InitialSystemContent != "" {
		messages = append([]types.Message{{
			Role:    "system",
			Content: t.InitialSystemContent,
		}}, messages...)
	}
	return messages
}
func getThread(t *Tzap) []types.Message {
	var messages []types.Message

	if t.Parent != nil {
		messages = GetThread(t.Parent)
	}

	if t.Message.Content == "" || t.Message.Role == "" {
		return messages
	}
	key, ok := t.Data["memory"].(string)
	if ok && key != "" {
		mV := Mem[key]
		if mV.Content != "" {
			message := types.Message{
				Role:    mV.Role,
				Content: mV.Content,
			}
			messages = append(messages, message)
		}
	}
	return append(messages, t.Message)
}
func (t *Tzap) StoreThread(filePath string) *ErrorTzap {
	messages := GetThread(t)
	jsonBytes, err := json.Marshal(messages)
	if err != nil {
		panic(fmt.Errorf("error storing thread: %w", err))
	}

	if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
		t.ErrorTzap(fmt.Errorf("StoreThread: error storing thread: %w", err))
	}

	return t.ErrorTzap(nil)
}
func (t *Tzap) LoadThread(filePath string) *ErrorTzap {
	var messages []types.Message
	err := json.Unmarshal([]byte(util.ReadFileP(filePath)), &messages)
	if err != nil {
		t.ErrorTzap(fmt.Errorf("error loading thread: %w", err))
	}
	return t.loadThread(messages).ErrorTzap(nil)
}

func (t *Tzap) loadThread(messages []types.Message) *Tzap {
	for _, message := range messages {
		if message.Role == openai.ChatMessageRoleSystem {
			t = t.AddSystemMessage(message.Content)
			continue
		}
		if message.Role == openai.ChatMessageRoleAssistant {
			t = t.AddAssistantMessage(message.Content)
			continue
		}
		if message.Role == openai.ChatMessageRoleUser {
			t = t.AddUserMessage(message.Content)
			continue
		}
	}
	return t
}
func (t *Tzap) storeThread(messages []types.Message) *Tzap {
	for _, message := range messages {
		if message.Role == openai.ChatMessageRoleSystem {
			t.AddSystemMessage(message.Content)
			continue
		}
		if message.Role == openai.ChatMessageRoleAssistant {
			t.AddAssistantMessage(message.Content)
			continue
		}
		if message.Role == openai.ChatMessageRoleUser {
			t.AddUserMessage(message.Content)
			continue
		}
	}
	return t
}

// Not accurate way of counting tokens, but an approximation.
func TruncateToMaxWords(messages []types.Message, wordLimit int) []types.Message {
	var result []types.Message
	wordCount := 0
	if wordLimit < 0 {
		panic(fmt.Sprintf("TruncateToMaxWords wordlimit is %d, set above 1, or 0 to allow unlimited until model fails", wordLimit))
	}
	if wordLimit == 0 {
		return messages
	}

	wordSplitter := regexp.MustCompile(`[\s,./()\-_]+`)
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		words := wordSplitter.Split(message.Content, -1)
		words = removeEmptyStrings(words)
		if wordCount+len(words) <= wordLimit {

			wordCount += len(words)
			result = append([]types.Message{message}, result...)
		} else {
			break
		}
	}

	return result
}

func removeEmptyStrings(strings []string) []string {
	var result []string
	for _, s := range strings {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}