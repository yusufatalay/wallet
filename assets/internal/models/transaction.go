package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TransactionStatus represents transaction's status.
type TransactionStatus string

const (
	StatusScheduled  TransactionStatus = "SCHEDULED"
	StatusProcessing TransactionStatus = "PROCESSING"
	StatusCompleted  TransactionStatus = "COMPLETED"
	StatusFailed     TransactionStatus = "FAILED"
)

// TransactionType represents transaction's type.
type TransactionType string

const (
	TypeDeposit  TransactionType = "DEPOSIT"
	TypeWithdraw TransactionType = "WITHDRAW"
	TypeTransfer TransactionType = "TRANSFER"
)

// TransferDetails, holds wallet transfer details.
type TransferDetails struct {
	FromWalletID    primitive.ObjectID `bson:"fromWalletId,omitempty"`
	ToWalletAddress string             `bson:"toWalletAddress,omitempty"`
	ToWalletNetwork string             `bson:"toWalletNetwork,omitempty"`
}

// Transaction represents a wallet transaction.
type Transaction struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Type      TransactionType    `bson:"type"`
	Status    TransactionStatus  `bson:"status"`
	AssetName string             `bson:"assetName"`
	Amount    float64            `bson:"amount"`

	TransferDetails `bson:",inline,omitempty"`

	// for Deposit/Withdraw
	WalletID primitive.ObjectID `bson:"walletId,omitempty"`

	ScheduledFor time.Time `bson:"scheduledFor,omitempty"`
	Timestamp    time.Time `bson:"timestamp"`
	ProcessNotes string    `bson:"processNotes,omitempty"`
}
