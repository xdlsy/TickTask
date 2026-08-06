package service

import (
	"time"
	"ticktask/internal/model"
	"ticktask/internal/repository"
)

const backupApp = "ticktask"

type DataService interface {
	Export() (*model.BackupEnvelope, error)
	PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error)
	ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error)
}

type dataService struct {
	repo repository.BackupRepository
}

func NewDataService(repo repository.BackupRepository) DataService {
	return &dataService{repo: repo}
}

func (s *dataService) Export() (*model.BackupEnvelope, error) {
	data, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	return &model.BackupEnvelope{
		App:           backupApp,
		SchemaVersion: model.BackupSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Data:          *data,
	}, nil
}

func (s *dataService) PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error) {
	return nil, nil // Task 5
}

func (s *dataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	return nil, nil // Task 6
}
