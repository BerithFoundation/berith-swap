// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: bers_swap_hist.sql

package mariadb

import (
	"context"
	"database/sql"
)

const createBersSwapHistory = `-- name: CreateBersSwapHistory :execresult
INSERT INTO bers_swap_hist(
    sender_tx_hash,
    receiver_tx_hash,
    amount,
    berith_address
) VALUES (
    ?,?,?,?
)
`

type CreateBersSwapHistoryParams struct {
	SenderTxHash   string `json:"sender_tx_hash"`
	ReceiverTxHash string `json:"receiver_tx_hash"`
	Amount         int64  `json:"amount"`
	BerithAddress  string `json:"berith_address"`
}

func (q *Queries) CreateBersSwapHistory(ctx context.Context, arg CreateBersSwapHistoryParams) (sql.Result, error) {
	return q.db.ExecContext(ctx, createBersSwapHistory,
		arg.SenderTxHash,
		arg.ReceiverTxHash,
		arg.Amount,
		arg.BerithAddress,
	)
}

const getBersSwapHistory = `-- name: GetBersSwapHistory :one
SELECT sender_tx_hash, receiver_tx_hash, berith_address, amount, created_at FROM bers_swap_hist
WHERE sender_tx_hash = ?
`

func (q *Queries) GetBersSwapHistory(ctx context.Context, senderTxHash string) (BersSwapHist, error) {
	row := q.db.QueryRowContext(ctx, getBersSwapHistory, senderTxHash)
	var i BersSwapHist
	err := row.Scan(
		&i.SenderTxHash,
		&i.ReceiverTxHash,
		&i.BerithAddress,
		&i.Amount,
		&i.CreatedAt,
	)
	return i, err
}

const getSwapHistByBerithAddress = `-- name: GetSwapHistByBerithAddress :many
SELECT sender_tx_hash, receiver_tx_hash, berith_address, amount, created_at FROM bers_swap_hist
WHERE berith_address = ?
`

func (q *Queries) GetSwapHistByBerithAddress(ctx context.Context, berithAddress string) ([]BersSwapHist, error) {
	rows, err := q.db.QueryContext(ctx, getSwapHistByBerithAddress, berithAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BersSwapHist{}
	for rows.Next() {
		var i BersSwapHist
		if err := rows.Scan(
			&i.SenderTxHash,
			&i.ReceiverTxHash,
			&i.BerithAddress,
			&i.Amount,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
