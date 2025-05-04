package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mudler/LocalAGI/pkg/xlog"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/sashabaranov/go-openai"
)

//... (rest of the content with memory-related changes)

// In the Agent struct, added:
// - ltm RAGDB field
// - Add methods for memory management

// Modified Run and consumeJob methods to handle memory retention

// Added new actions for memory management