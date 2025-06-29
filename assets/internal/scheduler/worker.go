package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yusufatalay/wallet/assets/internal/models"
	pb "github.com/yusufatalay/wallet/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Worker struct {
	Client       *mongo.Client
	Assets       *mongo.Collection
	Transactions *mongo.Collection
	WalletClient pb.WalletServiceClient
}

func (w *Worker) Start(ctx context.Context) {
	log.Println("[Cron Worker] starting...")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Cron Worker] stopping...")

			return
		case <-ticker.C:
			log.Println("[Cron Worker] checking for scheduled transactions...")
			w.processScheduledTransactions(ctx)
		}
	}
}

func (w *Worker) processScheduledTransactions(ctx context.Context) {
	filter := bson.M{
		"status":       models.StatusScheduled,
		"scheduledFor": bson.M{"$lte": time.Now()},
	}

	update := bson.M{"$set": bson.M{"status": models.StatusProcessing}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var tx models.Transaction
	err := w.Transactions.FindOneAndUpdate(ctx, filter, update, opts).Decode(&tx)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			log.Printf("[Cron Worker] failed to find transactions: %v", err)
		}

		return
	}

	log.Printf("[Cron Worker] job %s (%s) processing...", tx.ID.Hex(), tx.Type)

	var executionErr error
	switch tx.Type {
	case models.TypeTransfer:
		executionErr = w.executeTransfer(tx)
	case models.TypeDeposit:
		executionErr = w.executeDeposit(tx)
	case models.TypeWithdraw:
		executionErr = w.executeWithdraw(tx)
	default:
		executionErr = fmt.Errorf("[Cron Worker] unknown transaction type: %s", tx.Type)
	}

	finalStatus := models.StatusCompleted
	notes := "transaction successful"
	if executionErr != nil {
		finalStatus = models.StatusFailed
		notes = executionErr.Error()
		log.Printf("transaction %s failed: %v", tx.ID.Hex(), executionErr)
	}

	finalUpdate := bson.M{"$set": bson.M{"status": finalStatus, "processNotes": notes}}
	_, updateErr := w.Transactions.UpdateByID(context.Background(), tx.ID, finalUpdate)
	if updateErr != nil {
		log.Printf("[Cron Worker] failed to update transaction state: %v", updateErr)
	}
}

func (w *Worker) executeDeposit(tx models.Transaction) error {
	filter := bson.M{"walletId": tx.WalletID, "name": tx.AssetName}
	update := bson.M{"$inc": bson.M{"amount": tx.Amount}}
	opts := options.Update().SetUpsert(true)
	_, err := w.Assets.UpdateOne(context.Background(), filter, update, opts)

	return err
}

func (w *Worker) executeWithdraw(tx models.Transaction) error {
	session, err := w.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(context.Background())

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var asset models.Asset
		filter := bson.M{"walletId": tx.WalletID, "name": tx.AssetName}
		err := w.Assets.FindOne(sessCtx, filter).Decode(&asset)
		if err != nil || asset.Amount < tx.Amount {
			return nil, fmt.Errorf("insufficient funds: %v", err)
		}

		update := bson.M{"$inc": bson.M{"amount": -tx.Amount}}
		_, err = w.Assets.UpdateOne(sessCtx, filter, update)

		return nil, err
	}

	_, err = session.WithTransaction(context.Background(), callback)

	return err
}

func (w *Worker) executeTransfer(tx models.Transaction) error {
	session, err := w.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(context.Background())

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var sourceAsset models.Asset
		sourceFilter := bson.M{"walletId": tx.FromWalletID, "name": tx.AssetName}
		err := w.Assets.FindOne(sessCtx, sourceFilter).Decode(&sourceAsset)
		if err != nil || sourceAsset.Amount < tx.Amount {
			return nil, fmt.Errorf("insufficient funds: %v", err)
		}

		sourceUpdate := bson.M{"$inc": bson.M{"amount": -tx.Amount}}
		_, err = w.Assets.UpdateOne(sessCtx, sourceFilter, sourceUpdate)
		if err != nil {
			return nil, fmt.Errorf("failed to update source wallet: %w", err)
		}

		destWallet, err := w.WalletClient.FindWalletByAddress(sessCtx, &pb.FindWalletByAddressRequest{
			Address: tx.ToWalletAddress,
			Network: tx.ToWalletNetwork,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to find destination wallet via RPC: %w", err)
		}
		destWalletID, _ := primitive.ObjectIDFromHex(destWallet.WalletId)

		destAssetFilter := bson.M{"walletId": destWalletID, "name": tx.AssetName}
		destUpdate := bson.M{"$inc": bson.M{"amount": tx.Amount}}
		_, err = w.Assets.UpdateOne(sessCtx, destAssetFilter, destUpdate, options.Update().SetUpsert(true))
		if err != nil {
			return nil, fmt.Errorf("failed to update destination wallet: %w", err)
		}

		return nil, nil
	}

	_, err = session.WithTransaction(context.Background(), callback)

	return err
}
