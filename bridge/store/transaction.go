package store

import (
	"berith-swap/bridge/store/mariadb"
	"context"
)

func (s *Store) CreateSwapHistoryTx(ctx context.Context, arg mariadb.CreateBersSwapHistoryParams) error {
	err := s.execTx(ctx, func(q *mariadb.Queries) error {
		_, err := q.CreateBersSwapHistory(ctx, arg)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}
