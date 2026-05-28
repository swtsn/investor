package domain

import "fmt"

func (d Deployment) Validate(bucketType BucketType) error {
	switch bucketType {
	case BucketTypeFlat:
		if d.AllocationID != nil || d.Symbol != nil || d.Shares != nil || d.PricePerShare != nil {
			return fmt.Errorf("flat bucket deployments must not have allocation id, symbol, shares, or price per share")
		}
	case BucketTypeDiversified:
	default:
		return fmt.Errorf("unknown bucket type: %s", bucketType)
	}
	if (d.Shares == nil) != (d.PricePerShare == nil) {
		return fmt.Errorf("shares and price per share must be provided together")
	}
	if !d.Amount.IsPositive() {
		return fmt.Errorf("amount must be greater than zero")
	}
	if d.Shares != nil {
		derived := d.Shares.Mul(*d.PricePerShare)
		if !derived.Equal(d.Amount) {
			return fmt.Errorf("amount must equal shares * price per share: got %s, want %s", d.Amount, derived)
		}
	}
	return nil
}

func (c Contribution) Validate() error {
	switch c.Origination {
	case OriginationBudget, OriginationSlush:
		if c.BudgetEventID == nil {
			return fmt.Errorf("budget_event_id is required for origination=%s", c.Origination)
		}
	case OriginationReinvestment:
		if c.BudgetEventID != nil {
			return fmt.Errorf("budget_event_id must be nil for origination=reinvestment")
		}
	default:
		return fmt.Errorf("unknown origination type: %s", c.Origination)
	}
	if !c.Amount.IsPositive() {
		return fmt.Errorf("amount must be greater than zero")
	}
	return nil
}

func (ds DeploymentSource) Validate() error {
	if !ds.Amount.IsPositive() {
		return fmt.Errorf("amount must be greater than zero")
	}
	return nil
}
