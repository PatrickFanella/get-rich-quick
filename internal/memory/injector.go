package memory

import (
	"context"
	"log/slog"
	"strings"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

// Injector retrieves relevant memories and injects them into LLM messages.
type Injector struct {
	memoryRepo repository.MemoryRepository
	logger     *slog.Logger
}

// NewInjector creates an Injector with the given dependencies.
// A nil logger is replaced with slog.Default().
func NewInjector(memoryRepo repository.MemoryRepository, logger *slog.Logger) *Injector {
	if logger == nil {
		logger = slog.Default()
	}

	return &Injector{
		memoryRepo: memoryRepo,
		logger:     logger,
	}
}

// GetMemoryContext retrieves relevant memories for the given role and ticker and
// formats them for inclusion in an agent prompt.
func (i *Injector) GetMemoryContext(ctx context.Context, role domain.AgentRole, ticker string, limit int) (string, error) {
	memories, err := i.memoryRepo.Search(ctx, ticker, repository.MemorySearchFilter{
		AgentRole: role,
	}, limit, 0)
	if err != nil {
		return "", err
	}

	if len(memories) == 0 {
		return "", nil
	}

	var builder strings.Builder
	builder.WriteString("## Relevant Past Experience")
	for _, memory := range memories {
		builder.WriteString("\n- Situation: ")
		builder.WriteString(memory.Situation)
		builder.WriteString(", Lesson: ")
		builder.WriteString(memory.Recommendation)
	}

	return builder.String(), nil
}

// InjectIntoMessages appends memory context to the first system message when
// memory context is available.
func (i *Injector) InjectIntoMessages(messages []llm.Message, memoryContext string) []llm.Message {
	if memoryContext == "" {
		return messages
	}

	out := append([]llm.Message(nil), messages...)
	for idx := range out {
		if out[idx].Role != "system" {
			continue
		}

		if out[idx].Content == "" {
			out[idx].Content = memoryContext
		} else {
			out[idx].Content += "\n\n" + memoryContext
		}
		return out
	}

	return out
}
