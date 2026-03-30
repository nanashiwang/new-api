package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// CLI tools
	ClientClaudeCode = "claude-code"
	ClientCodexCLI   = "codex-cli"
	ClientGeminiCLI  = "gemini-cli"
	ClientDroidCLI   = "droid-cli"
	ClientAider      = "aider"

	// IDE plugins / editors
	ClientCursor      = "cursor"
	ClientWindsurf    = "windsurf"
	ClientCline       = "cline"
	ClientRooCode     = "roo-code"
	ClientContinue    = "continue"
	ClientTrae        = "trae"
	ClientZed         = "zed"
	ClientJetBrainsAI = "jetbrains"
	ClientAugment     = "augment"

	// Desktop / Web clients
	ClientCherryStudio = "cherry-studio"
	ClientChatBox      = "chatbox"
	ClientNextChat     = "nextchat"
	ClientLobeChat     = "lobechat"
	ClientOpenWebUI    = "open-webui"
	ClientLibreChat    = "librechat"
	ClientTypingMind   = "typingmind"
	ClientOpenCat      = "opencat"
	ClientBotGem       = "botgem"
	ClientJan          = "jan"

	// Fallback
	ClientAPIDirect = "api-direct"
)

type KnownClient struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AllKnownClients returns all known client definitions for frontend use.
func AllKnownClients() []KnownClient {
	return []KnownClient{
		// CLI tools
		{ID: ClientClaudeCode, Name: "Claude Code CLI"},
		{ID: ClientCodexCLI, Name: "Codex CLI"},
		{ID: ClientGeminiCLI, Name: "Gemini CLI"},
		{ID: ClientDroidCLI, Name: "Droid CLI"},
		{ID: ClientAider, Name: "Aider"},

		// IDE plugins / editors
		{ID: ClientCursor, Name: "Cursor"},
		{ID: ClientWindsurf, Name: "Windsurf"},
		{ID: ClientCline, Name: "Cline"},
		{ID: ClientRooCode, Name: "Roo Code"},
		{ID: ClientContinue, Name: "Continue"},
		{ID: ClientTrae, Name: "Trae"},
		{ID: ClientZed, Name: "Zed"},
		{ID: ClientJetBrainsAI, Name: "JetBrains AI"},
		{ID: ClientAugment, Name: "Augment"},

		// Desktop / Web clients
		{ID: ClientCherryStudio, Name: "Cherry Studio"},
		{ID: ClientChatBox, Name: "ChatBox"},
		{ID: ClientNextChat, Name: "NextChat"},
		{ID: ClientLobeChat, Name: "LobeChat"},
		{ID: ClientOpenWebUI, Name: "Open WebUI"},
		{ID: ClientLibreChat, Name: "LibreChat"},
		{ID: ClientTypingMind, Name: "TypingMind"},
		{ID: ClientOpenCat, Name: "OpenCat"},
		{ID: ClientBotGem, Name: "BotGem"},
		{ID: ClientJan, Name: "Jan"},

		// Fallback
		{ID: ClientAPIDirect, Name: "API 直连"},
	}
}

// DetectClient identifies the client tool from request headers using simple UA keyword matching.
func DetectClient(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ClientAPIDirect
	}

	ua := strings.ToLower(c.Request.UserAgent())

	// 1. High-confidence prefix matches (specific UA formats)
	if strings.HasPrefix(ua, "claude-cli/") {
		return ClientClaudeCode
	}
	if strings.HasPrefix(ua, "codex_cli_rs/") || strings.HasPrefix(ua, "codex_vscode/") {
		return ClientCodexCLI
	}
	if strings.HasPrefix(ua, "geminicli/") {
		return ClientGeminiCLI
	}
	if strings.HasPrefix(ua, "factory-cli/") {
		return ClientDroidCLI
	}

	// 2. Header-based detection for Droid CLI
	if factoryClient := strings.ToLower(c.GetHeader("x-factory-client")); factoryClient != "" {
		if strings.Contains(factoryClient, "droid") || strings.Contains(factoryClient, "factory-cli") {
			return ClientDroidCLI
		}
	}

	// 3. UA keyword contains matching (order matters: more specific first)
	if strings.Contains(ua, "claude-code") || strings.Contains(ua, "claudecode") {
		return ClientClaudeCode
	}
	if strings.Contains(ua, "codex_cli") || strings.Contains(ua, "codex_vscode") {
		return ClientCodexCLI
	}
	if strings.Contains(ua, "windsurf") || strings.Contains(ua, "codeium") {
		return ClientWindsurf
	}
	if strings.Contains(ua, "cursor") {
		return ClientCursor
	}
	if strings.Contains(ua, "aider") {
		return ClientAider
	}
	if strings.Contains(ua, "trae") {
		return ClientTrae
	}
	if strings.Contains(ua, "augment") {
		return ClientAugment
	}
	if strings.Contains(ua, "cherry") {
		return ClientCherryStudio
	}
	if strings.Contains(ua, "chatbox") {
		return ClientChatBox
	}
	if strings.Contains(ua, "nextchat") || strings.Contains(ua, "chatgpt-next-web") {
		return ClientNextChat
	}
	if strings.Contains(ua, "lobechat") {
		return ClientLobeChat
	}
	if strings.Contains(ua, "open-webui") {
		return ClientOpenWebUI
	}
	if strings.Contains(ua, "librechat") {
		return ClientLibreChat
	}
	if strings.Contains(ua, "typingmind") {
		return ClientTypingMind
	}
	if strings.Contains(ua, "opencat") {
		return ClientOpenCat
	}
	if strings.Contains(ua, "botgem") || strings.Contains(ua, "ama/") {
		return ClientBotGem
	}
	if strings.Contains(ua, "jan/") {
		return ClientJan
	}
	if strings.Contains(ua, "zed") {
		return ClientZed
	}
	if strings.Contains(ua, "jetbrains") || strings.Contains(ua, "intellij") {
		return ClientJetBrainsAI
	}
	if strings.Contains(ua, "continue") {
		return ClientContinue
	}

	// 4. x-title header fallback (VS Code extensions set this)
	xTitle := strings.ToLower(c.GetHeader("x-title"))
	if xTitle != "" {
		if strings.Contains(xTitle, "cline") {
			return ClientCline
		}
		if strings.Contains(xTitle, "roo") {
			return ClientRooCode
		}
		if strings.Contains(xTitle, "continue") {
			return ClientContinue
		}
		if strings.Contains(xTitle, "augment") {
			return ClientAugment
		}
	}

	return ClientAPIDirect
}
