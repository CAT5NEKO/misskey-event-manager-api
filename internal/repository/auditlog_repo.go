package repository

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"miSchedule/internal/model"
)

type AuditLogRepo struct {
	db *sql.DB
}

func NewAuditLogRepo(db *sql.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(log *model.AuditLog) error {
	var detailsBytes []byte
	if log.Details != nil {
		detailsBytes = log.Details
	}

	return r.db.QueryRow(
		`INSERT INTO audit_logs (actor_id, action, target_type, target_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		log.ActorID, log.Action, log.TargetType, log.TargetID,
		detailsBytes, log.IPAddress, log.UserAgent,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *AuditLogRepo) List(params model.AuditLogListParams) (*model.PaginatedAuditLogs, error) {
	baseWhere := ` WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if params.ActorID != nil {
		baseWhere += ` AND actor_id = $` + strconv.Itoa(argIdx)
		args = append(args, *params.ActorID)
		argIdx++
	}
	if params.Action != nil {
		baseWhere += ` AND action = $` + strconv.Itoa(argIdx)
		args = append(args, *params.Action)
		argIdx++
	}
	if params.TargetType != nil {
		baseWhere += ` AND target_type = $` + strconv.Itoa(argIdx)
		args = append(args, *params.TargetType)
		argIdx++
	}
	if params.From != nil {
		baseWhere += ` AND created_at >= $` + strconv.Itoa(argIdx)
		args = append(args, *params.From)
		argIdx++
	}
	if params.To != nil {
		baseWhere += ` AND created_at <= $` + strconv.Itoa(argIdx)
		args = append(args, *params.To)
		argIdx++
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM audit_logs` + baseWhere
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, err
	}

	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	offset := (params.Page - 1) * params.Limit

	selectQuery := `SELECT id, actor_id, action, target_type, target_id, details, ip_address, user_agent, created_at
		FROM audit_logs` + baseWhere + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, params.Limit, offset)

	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		var detailsBytes []byte
		if err := rows.Scan(&l.ID, &l.ActorID, &l.Action, &l.TargetType, &l.TargetID,
			&detailsBytes, &l.IPAddress, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, err
		}
		if len(detailsBytes) > 0 {
			l.Details = json.RawMessage(detailsBytes)
		}
		logs = append(logs, l)
	}

	return &model.PaginatedAuditLogs{
		Logs:       logs,
		TotalCount: total,
		Page:       params.Page,
		Limit:      params.Limit,
	}, nil
}

func (r *AuditLogRepo) DeleteOlderThan(before time.Time) error {
	_, err := r.db.Exec(`DELETE FROM audit_logs WHERE created_at < $1`, before)
	return err
}
