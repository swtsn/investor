package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func ptr[T any](v T) *T { return &v }

func TestDeploymentValidate(t *testing.T) {
	cases := []struct {
		name       string
		d          Deployment
		bucketType BucketType
		wantErr    bool
	}{
		{
			name:       "flat/valid amount-only",
			d:          Deployment{Amount: decimal.NewFromInt(100)},
			bucketType: BucketTypeFlat,
			wantErr:    false,
		},
		{
			name:       "flat/symbol present",
			d:          Deployment{Amount: decimal.NewFromInt(100), Symbol: ptr("AAPL")},
			bucketType: BucketTypeFlat,
			wantErr:    true,
		},
		{
			name:       "flat/shares present",
			d:          Deployment{Amount: decimal.NewFromInt(100), Shares: ptr(decimal.NewFromInt(5))},
			bucketType: BucketTypeFlat,
			wantErr:    true,
		},
		{
			name:       "flat/price_per_share present",
			d:          Deployment{Amount: decimal.NewFromInt(100), PricePerShare: ptr(decimal.NewFromInt(20))},
			bucketType: BucketTypeFlat,
			wantErr:    true,
		},
		{
			name:       "flat/allocation_id present",
			d:          Deployment{Amount: decimal.NewFromInt(100), AllocationID: ptr(int64(1))},
			bucketType: BucketTypeFlat,
			wantErr:    true,
		},
		{
			name:       "diversified/shares without price",
			d:          Deployment{Amount: decimal.NewFromInt(100), Shares: ptr(decimal.NewFromInt(5))},
			bucketType: BucketTypeDiversified,
			wantErr:    true,
		},
		{
			name:       "diversified/price without shares",
			d:          Deployment{Amount: decimal.NewFromInt(100), PricePerShare: ptr(decimal.NewFromInt(20))},
			bucketType: BucketTypeDiversified,
			wantErr:    true,
		},
		{
			name:       "diversified/valid amount-only",
			d:          Deployment{Amount: decimal.NewFromInt(100)},
			bucketType: BucketTypeDiversified,
			wantErr:    false,
		},
		{
			name: "diversified/shares and price correct amount",
			d: Deployment{
				Amount:        decimal.NewFromInt(100),
				Shares:        ptr(decimal.NewFromInt(5)),
				PricePerShare: ptr(decimal.NewFromInt(20)),
			},
			bucketType: BucketTypeDiversified,
			wantErr:    false,
		},
		{
			name: "diversified/shares and price wrong amount",
			d: Deployment{
				Amount:        decimal.NewFromInt(99),
				Shares:        ptr(decimal.NewFromInt(5)),
				PricePerShare: ptr(decimal.NewFromInt(20)),
			},
			bucketType: BucketTypeDiversified,
			wantErr:    true,
		},
		{
			name:       "amount zero",
			d:          Deployment{Amount: decimal.NewFromInt(0)},
			bucketType: BucketTypeFlat,
			wantErr:    true,
		},
		{
			name:       "amount negative",
			d:          Deployment{Amount: decimal.NewFromInt(-1)},
			bucketType: BucketTypeFlat,
			wantErr:    true,
		},
		{
			name:       "unknown bucket type",
			d:          Deployment{Amount: decimal.NewFromInt(100)},
			bucketType: BucketType("unknown"),
			wantErr:    true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.d.Validate(tc.bucketType)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestContributionValidate(t *testing.T) {
	eventID := int64(1)
	cases := []struct {
		name    string
		c       Contribution
		wantErr bool
	}{
		{
			name:    "budget/with event id",
			c:       Contribution{Amount: decimal.NewFromInt(100), Origination: OriginationBudget, BudgetEventID: &eventID},
			wantErr: false,
		},
		{
			name:    "budget/missing event id",
			c:       Contribution{Amount: decimal.NewFromInt(100), Origination: OriginationBudget},
			wantErr: true,
		},
		{
			name:    "slush/with event id",
			c:       Contribution{Amount: decimal.NewFromInt(100), Origination: OriginationSlush, BudgetEventID: &eventID},
			wantErr: false,
		},
		{
			name:    "slush/missing event id",
			c:       Contribution{Amount: decimal.NewFromInt(100), Origination: OriginationSlush},
			wantErr: true,
		},
		{
			name:    "reinvestment/no event id",
			c:       Contribution{Amount: decimal.NewFromInt(100), Origination: OriginationReinvestment},
			wantErr: false,
		},
		{
			name:    "reinvestment/event id present",
			c:       Contribution{Amount: decimal.NewFromInt(100), Origination: OriginationReinvestment, BudgetEventID: &eventID},
			wantErr: true,
		},
		{
			name:    "amount zero",
			c:       Contribution{Amount: decimal.NewFromInt(0), Origination: OriginationReinvestment},
			wantErr: true,
		},
		{
			name:    "amount negative",
			c:       Contribution{Amount: decimal.NewFromInt(-50), Origination: OriginationReinvestment},
			wantErr: true,
		},
		{
			name:    "unknown origination",
			c:       Contribution{Amount: decimal.NewFromInt(100), Origination: OriginationType("unknown")},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestDeploymentSourceValidate(t *testing.T) {
	cases := []struct {
		name    string
		ds      DeploymentSource
		wantErr bool
	}{
		{name: "positive", ds: DeploymentSource{Amount: decimal.NewFromInt(50)}, wantErr: false},
		{name: "zero", ds: DeploymentSource{Amount: decimal.NewFromInt(0)}, wantErr: true},
		{name: "negative", ds: DeploymentSource{Amount: decimal.NewFromInt(-10)}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ds.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
