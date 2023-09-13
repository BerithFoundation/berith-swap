-- name: CreateBersSwapHistory :execresult
INSERT INTO bers_swap_hist(
    sender_tx_hash,
    receiver_tx_hash,
    amount
) VALUES (
    ?,?,?
);

-- name: GetBersSwapHistory :one
SELECT * FROM bers_swap_hist
WHERE sender_tx_hash = ?;