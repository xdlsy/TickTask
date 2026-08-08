package repository

import (
	"testing"

	"ticktask/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentConversation{}, &model.AgentMessage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAgentRepo_CreateConversation(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)
	conv, err := repo.CreateConversation()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if conv.ID == "" {
		t.Fatal("id empty")
	}
	if conv.Title != "New Conversation" {
		t.Fatalf("default title: %q", conv.Title)
	}
}

func TestAgentRepo_AppendMessage_TitleFromFirstUser(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	longText := "今天有哪些没做完的任务需要顺延到明天并写日报总结"
	_, err := repo.AppendMessage(conv.ID, "user", longText, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	got, _ := repo.GetConversation(conv.ID)
	want := longText[:30]
	if got.Title != want {
		t.Fatalf("title = %q, want %q", got.Title, want)
	}
}

func TestAgentRepo_LoadRecentMessages(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	for i := 0; i < 25; i++ {
		repo.AppendMessage(conv.ID, "user", "msg", nil, nil, nil, nil)
	}
	msgs, err := repo.LoadRecentMessages(conv.ID, 20)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 20 {
		t.Fatalf("got %d, want 20", len(msgs))
	}
	if !msgs[0].CreatedAt.Before(msgs[19].CreatedAt) {
		t.Fatal("not ascending")
	}
}

func TestAgentRepo_DeleteConversation_Cascade(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	repo.AppendMessage(conv.ID, "user", "hi", nil, nil, nil, nil)
	if err := repo.DeleteConversation(conv.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := repo.GetConversation(conv.ID)
	if err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAgentRepo_UpdateMessageStatus(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	msgID, _ := repo.AppendMessage(conv.ID, "tool_call", "", strPtr("list_tasks"), nil, nil, nil)
	status := "succeeded"
	result := `{"tasks":[]}`
	if err := repo.UpdateMessage(msgID, &status, &result); err != nil {
		t.Fatalf("update: %v", err)
	}
}
