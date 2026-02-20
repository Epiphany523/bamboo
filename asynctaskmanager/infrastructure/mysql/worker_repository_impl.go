package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

// WorkerRepositoryImpl Worker 仓储 MySQL 实现
type WorkerRepositoryImpl struct {
	client *Client
}

// NewWorkerRepository 创建 Worker 仓储
func NewWorkerRepository(client *Client) repository.WorkerRepository {
	return &WorkerRepositoryImpl{client: client}
}

// Register 注册 Worker
func (r *WorkerRepositoryImpl) Register(ctx context.Context, worker *model.Worker) error {
	supportedTypes, err := json.Marshal(worker.SupportedTypes)
	if err != nil {
		return fmt.Errorf("marshal supported types failed: %w", err)
	}

	query := `INSERT INTO worker (worker_id, worker_name, address, status, capacity, current_load, supported_types, last_heartbeat, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE worker_name = VALUES(worker_name), address = VALUES(address), 
		status = VALUES(status), capacity = VALUES(capacity), supported_types = VALUES(supported_types), 
		last_heartbeat = VALUES(last_heartbeat), updated_at = VALUES(updated_at)`

	now := time.Now()
	_, err = r.client.db.ExecContext(ctx, query,
		worker.WorkerID,
		worker.WorkerName,
		worker.Address,
		worker.Status,
		worker.Capacity,
		worker.CurrentLoad,
		supportedTypes,
		worker.LastHeartbeat,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("register worker failed: %w", err)
	}

	return nil
}

// GetByID 根据ID查找 Worker
func (r *WorkerRepositoryImpl) GetByID(ctx context.Context, workerID string) (*model.Worker, error) {
	query := `SELECT id, worker_id, worker_name, address, status, capacity, current_load, supported_types, last_heartbeat, created_at, updated_at
		FROM worker WHERE worker_id = ?`

	row := r.client.db.QueryRowContext(ctx, query, workerID)

	worker := &model.Worker{}
	var supportedTypes []byte

	err := row.Scan(
		&worker.ID,
		&worker.WorkerID,
		&worker.WorkerName,
		&worker.Address,
		&worker.Status,
		&worker.Capacity,
		&worker.CurrentLoad,
		&supportedTypes,
		&worker.LastHeartbeat,
		&worker.CreatedAt,
		&worker.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("worker not found: %s", workerID)
	}
	if err != nil {
		return nil, fmt.Errorf("query worker failed: %w", err)
	}

	// 解析 supported_types
	if err := json.Unmarshal(supportedTypes, &worker.SupportedTypes); err != nil {
		return nil, fmt.Errorf("unmarshal supported types failed: %w", err)
	}

	return worker, nil
}

// Update 更新 Worker
func (r *WorkerRepositoryImpl) Update(ctx context.Context, worker *model.Worker) error {
	supportedTypes, err := json.Marshal(worker.SupportedTypes)
	if err != nil {
		return fmt.Errorf("marshal supported types failed: %w", err)
	}

	query := `UPDATE worker SET worker_name = ?, address = ?, status = ?, capacity = ?, 
		current_load = ?, supported_types = ?, last_heartbeat = ?, updated_at = ? WHERE worker_id = ?`

	_, err = r.client.db.ExecContext(ctx, query,
		worker.WorkerName,
		worker.Address,
		worker.Status,
		worker.Capacity,
		worker.CurrentLoad,
		supportedTypes,
		worker.LastHeartbeat,
		time.Now(),
		worker.WorkerID,
	)

	if err != nil {
		return fmt.Errorf("update worker failed: %w", err)
	}

	return nil
}

// Remove 移除 Worker
func (r *WorkerRepositoryImpl) Remove(ctx context.Context, workerID string) error {
	query := `DELETE FROM worker WHERE worker_id = ?`
	_, err := r.client.db.ExecContext(ctx, query, workerID)
	if err != nil {
		return fmt.Errorf("remove worker failed: %w", err)
	}
	return nil
}

// FindAll 查找所有 Worker
func (r *WorkerRepositoryImpl) FindAll(ctx context.Context) ([]*model.Worker, error) {
	query := `SELECT id, worker_id, worker_name, address, status, capacity, current_load, supported_types, last_heartbeat, created_at, updated_at
		FROM worker ORDER BY worker_id`

	rows, err := r.client.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all workers failed: %w", err)
	}
	defer rows.Close()

	return r.scanWorkers(rows)
}

// FindHealthy 查找健康的 Worker
func (r *WorkerRepositoryImpl) FindHealthy(ctx context.Context, timeout time.Duration) ([]*model.Worker, error) {
	query := `SELECT id, worker_id, worker_name, address, status, capacity, current_load, supported_types, last_heartbeat, created_at, updated_at
		FROM worker WHERE status = ? AND last_heartbeat >= ? ORDER BY current_load ASC`

	cutoffTime := time.Now().Add(-timeout)
	rows, err := r.client.db.QueryContext(ctx, query, model.WorkerOnline, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("query healthy workers failed: %w", err)
	}
	defer rows.Close()

	return r.scanWorkers(rows)
}

// FindByTaskType 根据任务类型查找支持的 Worker
func (r *WorkerRepositoryImpl) FindByTaskType(ctx context.Context, taskType string) ([]*model.Worker, error) {
	query := `SELECT id, worker_id, worker_name, address, status, capacity, current_load, supported_types, last_heartbeat, created_at, updated_at
		FROM worker WHERE status = ? AND JSON_CONTAINS(supported_types, ?) ORDER BY current_load ASC`

	taskTypeJSON := fmt.Sprintf(`"%s"`, taskType)
	rows, err := r.client.db.QueryContext(ctx, query, model.WorkerOnline, taskTypeJSON)
	if err != nil {
		return nil, fmt.Errorf("query workers by task type failed: %w", err)
	}
	defer rows.Close()

	return r.scanWorkers(rows)
}

// UpdateHeartbeat 更新心跳
func (r *WorkerRepositoryImpl) UpdateHeartbeat(ctx context.Context, workerID string) error {
	query := `UPDATE worker SET last_heartbeat = ? WHERE worker_id = ?`
	_, err := r.client.db.ExecContext(ctx, query, time.Now(), workerID)
	if err != nil {
		return fmt.Errorf("update heartbeat failed: %w", err)
	}
	return nil
}

// UpdateLoad 更新负载
func (r *WorkerRepositoryImpl) UpdateLoad(ctx context.Context, workerID string, load int) error {
	query := `UPDATE worker SET current_load = ? WHERE worker_id = ?`
	_, err := r.client.db.ExecContext(ctx, query, load, workerID)
	if err != nil {
		return fmt.Errorf("update load failed: %w", err)
	}
	return nil
}

// scanWorkers 扫描 Worker 列表
func (r *WorkerRepositoryImpl) scanWorkers(rows *sql.Rows) ([]*model.Worker, error) {
	workers := make([]*model.Worker, 0)

	for rows.Next() {
		worker := &model.Worker{}
		var supportedTypes []byte

		err := rows.Scan(
			&worker.ID,
			&worker.WorkerID,
			&worker.WorkerName,
			&worker.Address,
			&worker.Status,
			&worker.Capacity,
			&worker.CurrentLoad,
			&supportedTypes,
			&worker.LastHeartbeat,
			&worker.CreatedAt,
			&worker.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan worker failed: %w", err)
		}

		// 解析 supported_types
		if err := json.Unmarshal(supportedTypes, &worker.SupportedTypes); err != nil {
			return nil, fmt.Errorf("unmarshal supported types failed: %w", err)
		}

		workers = append(workers, worker)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return workers, nil
}
