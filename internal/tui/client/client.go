// Package client provides a typed wrapper around the InvestorService gRPC stubs.
// Views import this package and depend on the Client interface; they never
// import the generated proto packages directly.
package client

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	investorv1 "github.com/swtsn/investor/gen/investor/v1"
	"github.com/swtsn/investor/internal/domain"
)

// BucketDashboard is the per-bucket data for the Dashboard view.
type BucketDashboard struct {
	Bucket        domain.Bucket
	PoolBalance   decimal.Decimal
	Budget        decimal.Decimal
	Reinvestment  decimal.Decimal
	Slush         decimal.Decimal
	DeployedMonth decimal.Decimal
	ActualPct     decimal.Decimal
}

// BucketMonthData groups a bucket's events for a single month.
type BucketMonthData struct {
	Bucket        domain.Bucket
	Contributions []domain.Contribution
	Deployments   []domain.Deployment
}

// MonthSummary is the result of GetMonthSummary.
type MonthSummary struct {
	BudgetEvents []domain.BudgetEvent
	Buckets      []BucketMonthData
}

// DeployableSource is a contribution with remaining deployable balance.
type DeployableSource struct {
	Contribution domain.Contribution
	Remaining    decimal.Decimal
}

// BucketAllocation is one bucket's share of a budget split.
type BucketAllocation struct {
	Bucket domain.Bucket
	Amount decimal.Decimal
}

// AllocationSplit is one allocation's share of a deployment amount.
type AllocationSplit struct {
	Allocation domain.Allocation
	Amount     decimal.Decimal
}

// Client is the interface views use to talk to the server.
type Client interface {
	ListBuckets(ctx context.Context) ([]domain.Bucket, error)
	ListAllocations(ctx context.Context, bucketID int64) ([]domain.Allocation, error)
	GetDashboard(ctx context.Context) ([]BucketDashboard, error)
	GetMonthSummary(ctx context.Context, year, month int) (MonthSummary, error)
	ListDeployableSources(ctx context.Context, bucketID int64) ([]DeployableSource, error)
	PreviewBudget(ctx context.Context, total decimal.Decimal) ([]BucketAllocation, error)
	ApplyBudget(ctx context.Context, total decimal.Decimal, date time.Time) error
	PreviewDeployment(ctx context.Context, bucketID int64, amount decimal.Decimal) ([]AllocationSplit, error)
	RecordDeployment(ctx context.Context, d domain.Deployment, sources []domain.DeploymentSource) error
	RecordReinvestment(ctx context.Context, bucketID int64, amount decimal.Decimal, date time.Time) error
	CreateBucket(ctx context.Context, name string, bucketType domain.BucketType, targetPct decimal.Decimal) (domain.Bucket, error)
	UpdateBucket(ctx context.Context, id int64, name string, targetPct decimal.Decimal) (domain.Bucket, error)
	UpsertAllocation(ctx context.Context, bucketID int64, name string, targetPct decimal.Decimal) (domain.Allocation, error)
	DeleteAllocation(ctx context.Context, id int64) error
	Close() error
}

// grpcClient is the production implementation of Client.
type grpcClient struct {
	conn *grpc.ClientConn
	svc  investorv1.InvestorServiceClient
}

// New dials addr and returns a Client. Caller must call Close() when done.
func New(addr string) (Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &grpcClient{conn: conn, svc: investorv1.NewInvestorServiceClient(conn)}, nil
}

func (c *grpcClient) Close() error { return c.conn.Close() }

func (c *grpcClient) ListBuckets(ctx context.Context) ([]domain.Bucket, error) {
	resp, err := c.svc.ListBuckets(ctx, &investorv1.ListBucketsRequest{})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Bucket, len(resp.Buckets))
	for i, b := range resp.Buckets {
		result[i] = protoBucket(b)
	}
	return result, nil
}

func (c *grpcClient) ListAllocations(ctx context.Context, bucketID int64) ([]domain.Allocation, error) {
	resp, err := c.svc.ListAllocations(ctx, &investorv1.ListAllocationsRequest{BucketId: bucketID})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Allocation, len(resp.Allocations))
	for i, a := range resp.Allocations {
		result[i] = protoAllocation(a)
	}
	return result, nil
}

func (c *grpcClient) GetDashboard(ctx context.Context) ([]BucketDashboard, error) {
	resp, err := c.svc.GetDashboard(ctx, &investorv1.GetDashboardRequest{})
	if err != nil {
		return nil, err
	}
	result := make([]BucketDashboard, len(resp.Buckets))
	for i, b := range resp.Buckets {
		result[i] = BucketDashboard{
			Bucket: domain.Bucket{
				ID:        b.BucketId,
				Name:      b.Name,
				TargetPct: mustDecimal(b.TargetPct),
			},
			PoolBalance:   mustDecimal(b.PoolBalance),
			Budget:        mustDecimal(b.BreakdownBudget),
			Reinvestment:  mustDecimal(b.BreakdownReinvestment),
			Slush:         mustDecimal(b.BreakdownSlush),
			DeployedMonth: mustDecimal(b.DeployedThisMonth),
			ActualPct:     mustDecimal(b.ActualPct),
		}
	}
	return result, nil
}

func (c *grpcClient) GetMonthSummary(ctx context.Context, year, month int) (MonthSummary, error) {
	resp, err := c.svc.GetMonthSummary(ctx, &investorv1.GetMonthSummaryRequest{
		Year:  int32(year),
		Month: int32(month),
	})
	if err != nil {
		return MonthSummary{}, err
	}

	s := MonthSummary{
		BudgetEvents: make([]domain.BudgetEvent, len(resp.BudgetEvents)),
		Buckets:      make([]BucketMonthData, len(resp.Buckets)),
	}
	for i, e := range resp.BudgetEvents {
		s.BudgetEvents[i] = domain.BudgetEvent{
			ID:          e.Id,
			TotalAmount: mustDecimal(e.TotalAmount),
			Date:        e.Date.AsTime(),
		}
	}
	for i, b := range resp.Buckets {
		bmd := BucketMonthData{Bucket: protoBucket(b.Bucket)}
		for _, contrib := range b.Contributions {
			conv, err := protoContribution(contrib)
			if err != nil {
				return MonthSummary{}, err
			}
			bmd.Contributions = append(bmd.Contributions, conv)
		}
		for _, dep := range b.Deployments {
			conv, err := protoDeployment(dep)
			if err != nil {
				return MonthSummary{}, err
			}
			bmd.Deployments = append(bmd.Deployments, conv)
		}
		s.Buckets[i] = bmd
	}
	return s, nil
}

func (c *grpcClient) ListDeployableSources(ctx context.Context, bucketID int64) ([]DeployableSource, error) {
	resp, err := c.svc.ListDeployableSources(ctx, &investorv1.ListDeployableSourcesRequest{BucketId: bucketID})
	if err != nil {
		return nil, err
	}
	result := make([]DeployableSource, len(resp.Sources))
	for i, s := range resp.Sources {
		contrib, err := protoContribution(s.Contribution)
		if err != nil {
			return nil, err
		}
		result[i] = DeployableSource{
			Contribution: contrib,
			Remaining:    mustDecimal(s.Remaining),
		}
	}
	return result, nil
}

func (c *grpcClient) PreviewBudget(ctx context.Context, total decimal.Decimal) ([]BucketAllocation, error) {
	resp, err := c.svc.PreviewBudget(ctx, &investorv1.PreviewBudgetRequest{TotalAmount: total.String()})
	if err != nil {
		return nil, err
	}
	result := make([]BucketAllocation, len(resp.Allocations))
	for i, a := range resp.Allocations {
		result[i] = BucketAllocation{
			Bucket: protoBucket(a.Bucket),
			Amount: mustDecimal(a.Amount),
		}
	}
	return result, nil
}

func (c *grpcClient) ApplyBudget(ctx context.Context, total decimal.Decimal, date time.Time) error {
	_, err := c.svc.ApplyBudget(ctx, &investorv1.ApplyBudgetRequest{
		TotalAmount: total.String(),
		Date:        timestamppb.New(date),
	})
	return err
}

func (c *grpcClient) PreviewDeployment(ctx context.Context, bucketID int64, amount decimal.Decimal) ([]AllocationSplit, error) {
	resp, err := c.svc.PreviewDeployment(ctx, &investorv1.PreviewDeploymentRequest{
		BucketId: bucketID,
		Amount:   amount.String(),
	})
	if err != nil {
		return nil, err
	}
	result := make([]AllocationSplit, len(resp.Splits))
	for i, s := range resp.Splits {
		result[i] = AllocationSplit{
			Allocation: protoAllocation(s.Allocation),
			Amount:     mustDecimal(s.Amount),
		}
	}
	return result, nil
}

func (c *grpcClient) RecordDeployment(ctx context.Context, d domain.Deployment, sources []domain.DeploymentSource) error {
	req := &investorv1.RecordDeploymentRequest{
		BucketId:     d.BucketID,
		AllocationId: d.AllocationID,
		Symbol:       d.Symbol,
		Amount:       d.Amount.String(),
		Date:         timestamppb.New(d.Date),
	}
	if d.Shares != nil {
		s := d.Shares.String()
		req.Shares = &s
	}
	if d.PricePerShare != nil {
		s := d.PricePerShare.String()
		req.PricePerShare = &s
	}
	req.Sources = make([]*investorv1.DeploymentSource, len(sources))
	for i, s := range sources {
		req.Sources[i] = &investorv1.DeploymentSource{
			ContributionId: s.ContributionID,
			Amount:         s.Amount.String(),
		}
	}
	_, err := c.svc.RecordDeployment(ctx, req)
	return err
}

func (c *grpcClient) RecordReinvestment(ctx context.Context, bucketID int64, amount decimal.Decimal, date time.Time) error {
	_, err := c.svc.RecordReinvestment(ctx, &investorv1.RecordReinvestmentRequest{
		BucketId: bucketID,
		Amount:   amount.String(),
		Date:     timestamppb.New(date),
	})
	return err
}

func (c *grpcClient) CreateBucket(ctx context.Context, name string, bucketType domain.BucketType, targetPct decimal.Decimal) (domain.Bucket, error) {
	resp, err := c.svc.CreateBucket(ctx, &investorv1.CreateBucketRequest{
		Name:      name,
		Type:      string(bucketType),
		TargetPct: targetPct.String(),
	})
	if err != nil {
		return domain.Bucket{}, err
	}
	return protoBucket(resp.Bucket), nil
}

func (c *grpcClient) UpdateBucket(ctx context.Context, id int64, name string, targetPct decimal.Decimal) (domain.Bucket, error) {
	resp, err := c.svc.UpdateBucket(ctx, &investorv1.UpdateBucketRequest{
		Id:        id,
		Name:      name,
		TargetPct: targetPct.String(),
	})
	if err != nil {
		return domain.Bucket{}, err
	}
	return protoBucket(resp.Bucket), nil
}

func (c *grpcClient) UpsertAllocation(ctx context.Context, bucketID int64, name string, targetPct decimal.Decimal) (domain.Allocation, error) {
	resp, err := c.svc.UpsertAllocation(ctx, &investorv1.UpsertAllocationRequest{
		BucketId:  bucketID,
		Name:      name,
		TargetPct: targetPct.String(),
	})
	if err != nil {
		return domain.Allocation{}, err
	}
	return protoAllocation(resp.Allocation), nil
}

func (c *grpcClient) DeleteAllocation(ctx context.Context, id int64) error {
	_, err := c.svc.DeleteAllocation(ctx, &investorv1.DeleteAllocationRequest{Id: id})
	return err
}

// --- proto conversion helpers ---

func protoBucket(b *investorv1.Bucket) domain.Bucket {
	if b == nil {
		return domain.Bucket{}
	}
	return domain.Bucket{
		ID:        b.Id,
		Name:      b.Name,
		Type:      domain.BucketType(b.Type),
		TargetPct: mustDecimal(b.TargetPct),
	}
}

func protoAllocation(a *investorv1.Allocation) domain.Allocation {
	if a == nil {
		return domain.Allocation{}
	}
	return domain.Allocation{
		ID:        a.Id,
		BucketID:  a.BucketId,
		Name:      a.Name,
		TargetPct: mustDecimal(a.TargetPct),
	}
}

func protoContribution(p *investorv1.Contribution) (domain.Contribution, error) {
	if p == nil {
		return domain.Contribution{}, nil
	}
	amount, err := decimal.NewFromString(p.Amount)
	if err != nil {
		return domain.Contribution{}, err
	}
	return domain.Contribution{
		ID:            p.Id,
		BucketID:      p.BucketId,
		Amount:        amount,
		Origination:   domain.OriginationType(p.Origination),
		BudgetEventID: p.BudgetEventId,
		Date:          p.Date.AsTime(),
	}, nil
}

func protoDeployment(p *investorv1.Deployment) (domain.Deployment, error) {
	if p == nil {
		return domain.Deployment{}, nil
	}
	amount, err := decimal.NewFromString(p.Amount)
	if err != nil {
		return domain.Deployment{}, err
	}
	d := domain.Deployment{
		ID:           p.Id,
		BucketID:     p.BucketId,
		AllocationID: p.AllocationId,
		Symbol:       p.Symbol,
		Amount:       amount,
		Date:         p.Date.AsTime(),
	}
	if p.Shares != nil {
		v := mustDecimal(*p.Shares)
		d.Shares = &v
	}
	if p.PricePerShare != nil {
		v := mustDecimal(*p.PricePerShare)
		d.PricePerShare = &v
	}
	return d, nil
}

// mustDecimal parses a decimal string, returning zero on error.
func mustDecimal(s string) decimal.Decimal {
	v, _ := decimal.NewFromString(s)
	return v
}

// Compile-time check that grpcClient implements Client.
var _ Client = (*grpcClient)(nil)
