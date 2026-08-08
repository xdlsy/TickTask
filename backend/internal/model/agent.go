package model

import "time"

type AgentConversation struct {
	ID           string    `gorm:"primaryKey;type:text" json:"id"`
	Title        string    `gorm:"size:200" json:"title"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

type AgentMessage struct {
	ID             string    `gorm:"primaryKey;type:text" json:"id"`
	ConversationID string    `gorm:"index;type:text" json:"conversation_id"`
	Role           string    `gorm:"size:20" json:"role"`
	Content        string    `gorm:"type:text" json:"content"`
	ToolName       *string   `gorm:"size:50" json:"tool_name,omitempty"`
	ToolArgs       *string   `gorm:"type:text" json:"tool_args,omitempty"`
	ToolResult     *string   `gorm:"type:text" json:"tool_result,omitempty"`
	ToolStatus     *string   `gorm:"size:30" json:"tool_status,omitempty"`
	ToolCalls      *string   `gorm:"type:text" json:"tool_calls,omitempty"` // JSON []ai.ToolCall for role=assistant turns that requested tools
	ParentID       *string   `gorm:"type:text" json:"parent_id,omitempty"`
	CreatedAt      time.Time `gorm:"index" json:"created_at"`
}
