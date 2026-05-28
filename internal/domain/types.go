package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type BucketType string

const (
	BucketTypeFlat        BucketType = "flat"
	BucketTypeDiversified BucketType = "diversified"
)

type OriginationType string

const (
	OriginationBudget       OriginationType = "budget"
	OriginationReinvestment OriginationType = "reinvestment"
	OriginationSlush        OriginationType = "slush"
)

type Bucket struct {
	ID        int64
	Name      string
	Type      BucketType
	TargetPct decimal.Decimal
}

type Allocation struct {
	ID        int64
	BucketID  int64
	Name      string
	TargetPct decimal.Decimal
}

type BudgetEvent struct {
	ID          int64
	TotalAmount decimal.Decimal
	Date        time.Time
}

type Contribution struct {
	ID            int64
	BucketID      int64
	Amount        decimal.Decimal
	Origination   OriginationType
	BudgetEventID *int64
	Date          time.Time
}

type Deployment struct {
	ID            int64
	BucketID      int64
	AllocationID  *int64
	Symbol        *string
	Shares        *decimal.Decimal
	PricePerShare *decimal.Decimal
	Amount        decimal.Decimal
	Date          time.Time
}

type DeploymentSource struct {
	DeploymentID   int64
	ContributionID int64
	Amount         decimal.Decimal
}
