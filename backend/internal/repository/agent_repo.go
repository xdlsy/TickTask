package repository

import (
	"crypto/rand"
	"fmt"
	"time"

	"ticktask/internal/model"

	"gorm.io/gorm"
)

type AgentRepository interface {
	CreateConversation() (*model.AgentConversation, error)
	GetConversation(id string) (*model.AgentConversation, error)
	ListConversations(page, size int) ([]*model.AgentConversation, int, error)
	DeleteConversation(id string) error
	AppendMessage(convID, role, content string, toolName, toolArgs, toolResult, toolStatus *string) (string, error)
	LoadRecentMessages(convID string, limit int) ([]*model.AgentMessage, error)
	ListMessages(convID string) ([]*model.AgentMessage, error)
	UpdateMessage(id string, status, result *string) error
	GetMessage(id string) (*model.AgentMessage, error)
}

type agentRepo struct{ db *gorm.DB }

func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepo{db: db}
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (r *agentRepo) CreateConversation() (*model.AgentConversation, error) {
	conv := &model.AgentConversation{
		ID:        newUUID(),
		Title:     "New Conversation",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.db.Create(conv).Error; err != nil {
		return nil, err
	}
	return conv, nil
}

func (r *agentRepo) GetConversation(id string) (*model.AgentConversation, error) {
	var c model.AgentConversation
	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *agentRepo) ListConversations(page, size int) ([]*model.AgentConversation, int, error) {
	var items []*model.AgentConversation
	var total int64
	r.db.Model(&model.AgentConversation{}).Count(&total)
	off := (page - 1) * size
	if err := r.db.Order("updated_at DESC").Offset(off).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, int(total), nil
}

func (r *agentRepo) DeleteConversation(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("conversation_id = ?", id).Delete(&model.AgentMessage{})
		return tx.Where("id = ?", id).Delete(&model.AgentConversation{}).Error
	})
}

func (r *agentRepo) AppendMessage(convID, role, content string, toolName, toolArgs, toolResult, toolStatus *string) (string, error) {
	msg := &model.AgentMessage{
		ID:             newUUID(),
		ConversationID: convID,
		Role:           role,
		Content:        content,
		ToolName:       toolName,
		ToolArgs:       toolArgs,
		ToolResult:     toolResult,
		ToolStatus:     toolStatus,
		CreatedAt:      time.Now(),
	}
	if err := r.db.Create(msg).Error; err != nil {
		return "", err
	}
	r.db.Model(&model.AgentConversation{}).Where("id = ?", convID).
		UpdateColumns(map[string]any{"updated_at": time.Now(), "message_count": gorm.Expr("message_count + 1")})
	if role == "user" && len(content) > 0 {
		title := content
		if len(title) > 30 {
			title = title[:30]
		}
		r.db.Model(&model.AgentConversation{}).Where("id = ?", convID).Update("title", title)
	}
	return msg.ID, nil
}

func (r *agentRepo) LoadRecentMessages(convID string, limit int) ([]*model.AgentMessage, error) {
	var msgs []*model.AgentMessage
	sub := r.db.Model(&model.AgentMessage{}).Where("conversation_id = ?", convID).
		Order("created_at DESC").Limit(limit)
	if err := r.db.Table("(?) AS u", sub).Order("u.created_at ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

func (r *agentRepo) ListMessages(convID string) ([]*model.AgentMessage, error) {
	var msgs []*model.AgentMessage
	if err := r.db.Where("conversation_id = ?", convID).Order("created_at ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

func (r *agentRepo) UpdateMessage(id string, status, result *string) error {
	updates := map[string]any{}
	if status != nil {
		updates["tool_status"] = *status
	}
	if result != nil {
		updates["tool_result"] = *result
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&model.AgentMessage{}).Where("id = ?", id).Updates(updates).Error
}

func (r *agentRepo) GetMessage(id string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	if err := r.db.First(&m, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}
