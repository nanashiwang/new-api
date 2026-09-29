package dto

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
)

// Keep custom JSON methods on request-only wrappers. Message is also embedded
// in response choices; methods on Message would hide sibling response fields.
type openAIRequestMessage Message

func (r *GeneralOpenAIRequest) UnmarshalJSON(data []byte) error {
	type requestAlias GeneralOpenAIRequest
	var decoded requestAlias
	wire := struct {
		*requestAlias
		Messages []openAIRequestMessage `json:"messages"`
	}{requestAlias: &decoded}
	if err := common.Unmarshal(data, &wire); err != nil {
		return err
	}
	if wire.Messages != nil {
		decoded.Messages = make([]Message, len(wire.Messages))
		for i, message := range wire.Messages {
			decoded.Messages[i] = Message(message)
		}
	}
	*r = GeneralOpenAIRequest(decoded)
	return nil
}

func (r GeneralOpenAIRequest) MarshalJSON() ([]byte, error) {
	type requestAlias GeneralOpenAIRequest
	var messages []openAIRequestMessage
	if r.Messages != nil {
		messages = make([]openAIRequestMessage, len(r.Messages))
		for i, message := range r.Messages {
			messages[i] = openAIRequestMessage(message)
		}
	}
	return common.Marshal(struct {
		requestAlias
		Messages []openAIRequestMessage `json:"messages,omitempty"`
	}{requestAlias: requestAlias(r), Messages: messages})
}

func (m *openAIRequestMessage) UnmarshalJSON(data []byte) error {
	type messageAlias Message
	var decoded messageAlias
	wire := struct {
		*messageAlias
		Content json.RawMessage `json:"content"`
	}{messageAlias: &decoded}
	if err := common.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.ContentPresent = len(wire.Content) > 0
	if decoded.ContentPresent {
		if err := common.Unmarshal(wire.Content, &decoded.Content); err != nil {
			return err
		}
	}
	*m = openAIRequestMessage(decoded)
	return nil
}

func (m openAIRequestMessage) MarshalJSON() ([]byte, error) {
	type messageAlias Message
	if len(m.Tools) > 0 && m.Content == nil && !m.ContentPresent {
		// Preserve omitted content without inventing null. Official KVV also
		// uses an explicit empty string, which must be kept verbatim below.
		// Explicit null is preserved for upstream validation, not normalized.
		return common.Marshal(struct {
			messageAlias
			Content json.RawMessage `json:"content,omitempty"`
		}{messageAlias: messageAlias(m)})
	}
	return common.Marshal(messageAlias(m))
}
