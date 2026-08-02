package repository

import (
	"ticktask/internal/model"

	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(task *model.Task) error
	GetByID(id string) (*model.Task, error)
	GetAll() ([]model.Task, error)
	GetByQuadrant(quadrant model.Quadrant) ([]model.Task, error)
	GetByStatus(status model.TaskStatus) ([]model.Task, error)
	Update(task *model.Task) error
	Delete(id string) error
	GetAllByQuadrant() (map[model.Quadrant][]model.Task, error)
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(task *model.Task) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) GetByID(id string) (*model.Task, error) {
	var task model.Task
	err := r.db.Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) GetAll() ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) GetByQuadrant(quadrant model.Quadrant) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.Where("quadrant = ?", quadrant).
		Where("status != ?", model.StatusCancelled).
		Order("quadrant, `order` ASC, created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) GetByStatus(status model.TaskStatus) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.Where("status = ?", status).
		Order("quadrant, `order` ASC, created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) Update(task *model.Task) error {
	return r.db.Save(task).Error
}

func (r *taskRepository) Delete(id string) error {
	return r.db.Delete(&model.Task{}, "id = ?", id).Error
}

func (r *taskRepository) GetAllByQuadrant() (map[model.Quadrant][]model.Task, error) {
	var tasks []model.Task
	err := r.db.Where("status != ?", model.StatusCancelled).
		Order("quadrant, `order` ASC, created_at DESC").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	result := make(map[model.Quadrant][]model.Task)
	for _, task := range tasks {
		result[task.Quadrant] = append(result[task.Quadrant], task)
	}

	// 确保四个象限都存在
	for q := model.Quadrant1; q <= model.Quadrant4; q++ {
		if _, ok := result[q]; !ok {
			result[q] = []model.Task{}
		}
	}

	return result, nil
}
