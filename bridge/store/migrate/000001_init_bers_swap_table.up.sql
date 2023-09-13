CREATE TABLE `bers_swap_hist` (
  `sender_tx_hash` varchar(255) PRIMARY KEY,
  `receiver_tx_hash` varchar(255) NOT NULL,
  `amount` bigint NOT NULL,
  `created_at` timestamp DEFAULT (now())
);