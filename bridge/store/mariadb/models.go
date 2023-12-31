// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0

package mariadb

import (
	"database/sql"
)

type BersSwapHist struct {
	SenderTxHash   string       `json:"sender_tx_hash"`
	ReceiverTxHash string       `json:"receiver_tx_hash"`
	BerithAddress  string       `json:"berith_address"`
	Amount         int64        `json:"amount"`
	CreatedAt      sql.NullTime `json:"created_at"`
}
