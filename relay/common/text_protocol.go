package common

import (
	"fmt"

	"github.com/QuantumNous/new-api/types"
)

type TextProtocolSet uint8

const (
	textProtocolOpenAI TextProtocolSet = 1 << iota
	textProtocolClaude
	textProtocolGemini
	textProtocolOpenAIResponses
)

type TextProtocolConverter string

const (
	TextProtocolConverterNone               TextProtocolConverter = ""
	TextProtocolConverterClaudeToOpenAIChat TextProtocolConverter = "claude_messages_to_openai_chat"
)

type TextProtocolPlan struct {
	IncomingFormat types.RelayFormat
	UpstreamFormat types.RelayFormat
	Converter      TextProtocolConverter
}

type textProtocolConversion struct {
	From      types.RelayFormat
	To        types.RelayFormat
	Converter TextProtocolConverter
}

var textProtocolConversions = []textProtocolConversion{
	{
		From:      types.RelayFormatClaude,
		To:        types.RelayFormatOpenAI,
		Converter: TextProtocolConverterClaudeToOpenAIChat,
	},
}

func NewTextProtocolSet(formats ...types.RelayFormat) TextProtocolSet {
	var set TextProtocolSet
	for _, format := range formats {
		set |= textProtocolBit(format)
	}
	return set
}

func (s TextProtocolSet) Supports(format types.RelayFormat) bool {
	bit := textProtocolBit(format)
	return bit != 0 && s&bit != 0
}

func (p TextProtocolPlan) RequiresConversion() bool {
	return p.Converter != TextProtocolConverterNone && p.IncomingFormat != p.UpstreamFormat
}

func (p TextProtocolPlan) ValidatePassThrough(enabled bool) error {
	if enabled && p.RequiresConversion() {
		return fmt.Errorf("pass-through request body is incompatible with protocol conversion %s -> %s", p.IncomingFormat, p.UpstreamFormat)
	}
	return nil
}

func ResolveTextProtocolPlan(incomingFormat types.RelayFormat, nativeFormats TextProtocolSet) (TextProtocolPlan, error) {
	if incomingFormat == "" {
		return TextProtocolPlan{}, fmt.Errorf("incoming text protocol is required")
	}
	if nativeFormats.Supports(incomingFormat) {
		return TextProtocolPlan{
			IncomingFormat: incomingFormat,
			UpstreamFormat: incomingFormat,
		}, nil
	}
	for _, conversion := range textProtocolConversions {
		if conversion.From == incomingFormat && nativeFormats.Supports(conversion.To) {
			return TextProtocolPlan{
				IncomingFormat: incomingFormat,
				UpstreamFormat: conversion.To,
				Converter:      conversion.Converter,
			}, nil
		}
	}
	return TextProtocolPlan{}, fmt.Errorf("no safe text protocol route from %s to the selected channel", incomingFormat)
}

func (info *RelayInfo) PrepareTextProtocolPlan() error {
	if info == nil {
		return nil
	}
	info.TextProtocolPlan = nil
	info.FinalRequestRelayFormat = ""
	info.RequestConversionChain = nil
	info.InitRequestConversionChain()
	if info.ChannelMeta == nil || info.NativeTextFormats == 0 {
		return nil
	}
	plan, err := ResolveTextProtocolPlan(info.RelayFormat, info.NativeTextFormats)
	if err != nil {
		return err
	}
	info.TextProtocolPlan = &plan
	return nil
}

func (info *RelayInfo) CommitTextProtocolPlan() {
	if info == nil || info.TextProtocolPlan == nil {
		return
	}
	info.AppendRequestConversion(info.TextProtocolPlan.UpstreamFormat)
	info.FinalRequestRelayFormat = info.TextProtocolPlan.UpstreamFormat
}

func textProtocolBit(format types.RelayFormat) TextProtocolSet {
	switch format {
	case types.RelayFormatOpenAI:
		return textProtocolOpenAI
	case types.RelayFormatClaude:
		return textProtocolClaude
	case types.RelayFormatGemini:
		return textProtocolGemini
	case types.RelayFormatOpenAIResponses:
		return textProtocolOpenAIResponses
	default:
		return 0
	}
}
