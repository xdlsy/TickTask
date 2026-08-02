package repository

import "errors"

// 资源不存在
var ErrNotFound = errors.New("resource not found")

// WorkItem 不存在
var ErrItemNotFound = errors.New("work item not found")

// WorkItem 不能通过 quick-entry 接口修改（如 source='ai' 的条目）
var ErrItemNotEditable = errors.New("work item is not editable via this endpoint")
