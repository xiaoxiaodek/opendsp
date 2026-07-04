package data

import (
	"context"
	"fmt"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
)

type balanceRepo struct {
	data *Data
}

func NewBalanceRepo(data *Data) biz.BalanceRepo {
	return &balanceRepo{data: data}
}

func (r *balanceRepo) GetBalance(ctx context.Context, advertiserID int64) (float64, float64, error) {
	row, err := r.data.Queries.GetAdvertiserBalance(ctx, advertiserID)
	if err != nil {
		return 0, 0, fmt.Errorf("get balance: %w", err)
	}
	return row.Balance, row.CreditLimit, nil
}

func (r *balanceRepo) Recharge(ctx context.Context, advertiserID int64, amount float64, description string, operatorID *int64) (*biz.BalanceTransaction, error) {
	balanceBefore, _, err := r.GetBalance(ctx, advertiserID)
	if err != nil {
		return nil, err
	}

	balanceAfter := balanceBefore + amount
	balanceNumeric := float64ToNumeric(&balanceAfter)

	newBalance, err := r.data.Queries.RechargeAdvertiser(ctx, &dbsqlc.RechargeAdvertiserParams{
		ID:      advertiserID,
		Balance: balanceNumeric,
	})
	if err != nil {
		return nil, fmt.Errorf("recharge: %w", err)
	}
	balanceAfterVal := numericToFloat64Val(newBalance)

	// Sync balance to Redis so the ad-server budget guard can read it
	balanceKey := fmt.Sprintf("balance:%d", advertiserID)
	cents := int64(balanceAfterVal * 100) // store in 分 (cents)
	if r.data.Rdb != nil {
		if err := r.data.Rdb.Set(ctx, balanceKey, cents, 0).Err(); err != nil {
			return nil, fmt.Errorf("sync balance to redis: %w", err)
		}
	}

	txType := biz.TxTypeRecharge

	err = r.data.Queries.CreateBalanceTransaction(ctx, &dbsqlc.CreateBalanceTransactionParams{
		AdvertiserID:  advertiserID,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfterVal,
		TxType:        txType,
		Description:   &description,
		OperatorID:    operatorID,
	})
	if err != nil {
		return nil, fmt.Errorf("create tx: %w", err)
	}

	return &biz.BalanceTransaction{
		AdvertiserID:  advertiserID,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfterVal,
		TxType:        txType,
		Description:   &description,
	}, nil
}

func (r *balanceRepo) ListTransactions(ctx context.Context, advertiserID int64, page, pageSize int32) ([]biz.BalanceTransaction, int64, error) {
	total, err := r.data.Queries.CountBalanceTransactions(ctx, advertiserID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.data.Queries.ListBalanceTransactions(ctx, &dbsqlc.ListBalanceTransactionsParams{
		AdvertiserID: advertiserID,
		Limit:        pageSize,
		Offset:       offset,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]biz.BalanceTransaction, len(rows))
	for i, row := range rows {
		result[i] = biz.BalanceTransaction{
			ID:            row.ID,
			AdvertiserID:  row.AdvertiserID,
			Amount:        row.Amount,
			BalanceBefore: row.BalanceBefore,
			BalanceAfter:  row.BalanceAfter,
			TxType:        row.TxType,
			Description:   row.Description,
			OperatorID:    row.OperatorID,
			CreatedAt:     row.CreatedAt.Time,
		}
	}
	return result, total, nil
}
