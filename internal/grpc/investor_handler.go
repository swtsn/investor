// Package grpc implements the InvestorService gRPC handler.
package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/shopspring/decimal"
	investorv1 "github.com/swtsn/investor/gen/investor/v1"
	"github.com/swtsn/investor/internal/domain"
	"github.com/swtsn/investor/internal/service"
)

// InvestorHandler implements investorv1.InvestorServiceServer.
type InvestorHandler struct {
	investorv1.UnimplementedInvestorServiceServer
	budget *service.BudgetService
	deploy *service.DeploymentService
	pool   *service.PoolService
}

// NewInvestorHandler wires a handler backed by the given services.
func NewInvestorHandler(budget *service.BudgetService, deploy *service.DeploymentService, pool *service.PoolService) *InvestorHandler {
	return &InvestorHandler{budget: budget, deploy: deploy, pool: pool}
}

func toGRPCError(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return status.Error(codes.NotFound, "not found")
	}
	return status.Errorf(codes.Internal, "%v", err)
}

// --- proto conversion helpers ---

func bucketToProto(b domain.Bucket) *investorv1.Bucket {
	return &investorv1.Bucket{
		Id:        b.ID,
		Name:      b.Name,
		Type:      string(b.Type),
		TargetPct: b.TargetPct.String(),
	}
}

func allocationToProto(a domain.Allocation) *investorv1.Allocation {
	return &investorv1.Allocation{
		Id:        a.ID,
		BucketId:  a.BucketID,
		Name:      a.Name,
		TargetPct: a.TargetPct.String(),
	}
}

func contributionToProto(c domain.Contribution) *investorv1.Contribution {
	return &investorv1.Contribution{
		Id:            c.ID,
		BucketId:      c.BucketID,
		Amount:        c.Amount.String(),
		Origination:   string(c.Origination),
		BudgetEventId: c.BudgetEventID,
		Date:          timestamppb.New(c.Date),
	}
}

func deploymentToProto(d domain.Deployment) *investorv1.Deployment {
	p := &investorv1.Deployment{
		Id:           d.ID,
		BucketId:     d.BucketID,
		AllocationId: d.AllocationID,
		Symbol:       d.Symbol,
		Amount:       d.Amount.String(),
		Date:         timestamppb.New(d.Date),
	}
	if d.Shares != nil {
		s := d.Shares.String()
		p.Shares = &s
	}
	if d.PricePerShare != nil {
		s := d.PricePerShare.String()
		p.PricePerShare = &s
	}
	return p
}

// --- RPC implementations ---

func (h *InvestorHandler) ListBuckets(ctx context.Context, _ *investorv1.ListBucketsRequest) (*investorv1.ListBucketsResponse, error) {
	buckets, err := h.pool.ListBuckets(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	resp := &investorv1.ListBucketsResponse{Buckets: make([]*investorv1.Bucket, len(buckets))}
	for i, b := range buckets {
		resp.Buckets[i] = bucketToProto(b)
	}
	return resp, nil
}

func (h *InvestorHandler) ListAllocations(ctx context.Context, req *investorv1.ListAllocationsRequest) (*investorv1.ListAllocationsResponse, error) {
	allocs, err := h.pool.ListAllocations(ctx, req.BucketId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	resp := &investorv1.ListAllocationsResponse{Allocations: make([]*investorv1.Allocation, len(allocs))}
	for i, a := range allocs {
		resp.Allocations[i] = allocationToProto(a)
	}
	return resp, nil
}

func (h *InvestorHandler) GetDashboard(ctx context.Context, _ *investorv1.GetDashboardRequest) (*investorv1.GetDashboardResponse, error) {
	data, err := h.pool.GetDashboard(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}

	var total decimal.Decimal
	for _, d := range data {
		total = total.Add(d.PoolBalance)
	}

	hundred := decimal.NewFromInt(100)
	resp := &investorv1.GetDashboardResponse{}
	for _, d := range data {
		var actualPct decimal.Decimal
		if total.IsPositive() {
			actualPct = d.PoolBalance.Mul(hundred).Div(total).Round(1)
		}
		resp.Buckets = append(resp.Buckets, &investorv1.BucketDashboard{
			BucketId:              d.Bucket.ID,
			Name:                  d.Bucket.Name,
			TargetPct:             d.Bucket.TargetPct.String(),
			PoolBalance:           d.PoolBalance.String(),
			DeployedThisMonth:     d.DeployedMonth.String(),
			BreakdownBudget:       d.Breakdown.Budget.String(),
			BreakdownReinvestment: d.Breakdown.Reinvestment.String(),
			BreakdownSlush:        d.Breakdown.Slush.String(),
			ActualPct:             actualPct.String(),
		})
	}
	return resp, nil
}

func (h *InvestorHandler) GetMonthSummary(ctx context.Context, req *investorv1.GetMonthSummaryRequest) (*investorv1.GetMonthSummaryResponse, error) {
	summary, err := h.pool.GetMonthSummary(ctx, int(req.Year), int(req.Month))
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := &investorv1.GetMonthSummaryResponse{}
	for _, e := range summary.BudgetEvents {
		resp.BudgetEvents = append(resp.BudgetEvents, &investorv1.BudgetEvent{
			Id:          e.ID,
			TotalAmount: e.TotalAmount.String(),
			Date:        timestamppb.New(e.Date),
		})
	}
	for _, bs := range summary.Buckets {
		bmd := &investorv1.BucketMonthData{Bucket: bucketToProto(bs.Bucket)}
		for _, c := range bs.Contributions {
			bmd.Contributions = append(bmd.Contributions, contributionToProto(c))
		}
		for _, d := range bs.Deployments {
			bmd.Deployments = append(bmd.Deployments, deploymentToProto(d))
		}
		resp.Buckets = append(resp.Buckets, bmd)
	}
	return resp, nil
}

func (h *InvestorHandler) ListDeployableSources(ctx context.Context, req *investorv1.ListDeployableSourcesRequest) (*investorv1.ListDeployableSourcesResponse, error) {
	sources, err := h.pool.ListDeployableSources(ctx, req.BucketId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	resp := &investorv1.ListDeployableSourcesResponse{}
	for _, s := range sources {
		resp.Sources = append(resp.Sources, &investorv1.DeployableSource{
			Contribution: contributionToProto(s.Contribution),
			Remaining:    s.Remaining.String(),
		})
	}
	return resp, nil
}

func (h *InvestorHandler) PreviewBudget(ctx context.Context, req *investorv1.PreviewBudgetRequest) (*investorv1.PreviewBudgetResponse, error) {
	totalAmount, err := decimal.NewFromString(req.TotalAmount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid total_amount: %v", err)
	}
	allocs, err := h.budget.PreviewBudget(ctx, totalAmount)
	if err != nil {
		return nil, toGRPCError(err)
	}
	resp := &investorv1.PreviewBudgetResponse{}
	for _, a := range allocs {
		resp.Allocations = append(resp.Allocations, &investorv1.BucketAllocation{
			Bucket: bucketToProto(a.Bucket),
			Amount: a.Amount.String(),
		})
	}
	return resp, nil
}

func (h *InvestorHandler) ApplyBudget(ctx context.Context, req *investorv1.ApplyBudgetRequest) (*investorv1.ApplyBudgetResponse, error) {
	if req.Date == nil {
		return nil, status.Error(codes.InvalidArgument, "date is required")
	}
	totalAmount, err := decimal.NewFromString(req.TotalAmount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid total_amount: %v", err)
	}
	event, err := h.budget.ApplyBudget(ctx, totalAmount, req.Date.AsTime())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &investorv1.ApplyBudgetResponse{
		Event: &investorv1.BudgetEvent{
			Id:          event.ID,
			TotalAmount: event.TotalAmount.String(),
			Date:        timestamppb.New(event.Date),
		},
	}, nil
}

func (h *InvestorHandler) PreviewDeployment(ctx context.Context, req *investorv1.PreviewDeploymentRequest) (*investorv1.PreviewDeploymentResponse, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	splits, err := h.deploy.PreviewDeployment(ctx, req.BucketId, amount)
	if err != nil {
		return nil, toGRPCError(err)
	}
	resp := &investorv1.PreviewDeploymentResponse{}
	for _, s := range splits {
		resp.Splits = append(resp.Splits, &investorv1.AllocationSplit{
			Allocation: allocationToProto(s.Allocation),
			Amount:     s.Amount.String(),
		})
	}
	return resp, nil
}

func (h *InvestorHandler) RecordDeployment(ctx context.Context, req *investorv1.RecordDeploymentRequest) (*investorv1.RecordDeploymentResponse, error) {
	if req.Date == nil {
		return nil, status.Error(codes.InvalidArgument, "date is required")
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	d := domain.Deployment{
		BucketID:     req.BucketId,
		AllocationID: req.AllocationId,
		Symbol:       req.Symbol,
		Amount:       amount,
		Date:         req.Date.AsTime(),
	}
	if req.Shares != nil {
		v, err := decimal.NewFromString(*req.Shares)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid shares: %v", err)
		}
		d.Shares = &v
	}
	if req.PricePerShare != nil {
		v, err := decimal.NewFromString(*req.PricePerShare)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid price_per_share: %v", err)
		}
		d.PricePerShare = &v
	}

	sources := make([]domain.DeploymentSource, len(req.Sources))
	for i, s := range req.Sources {
		a, err := decimal.NewFromString(s.Amount)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid source amount: %v", err)
		}
		sources[i] = domain.DeploymentSource{ContributionID: s.ContributionId, Amount: a}
	}

	result, err := h.deploy.RecordDeployment(ctx, d, sources)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &investorv1.RecordDeploymentResponse{Deployment: deploymentToProto(result)}, nil
}

func (h *InvestorHandler) RecordReinvestment(ctx context.Context, req *investorv1.RecordReinvestmentRequest) (*investorv1.RecordReinvestmentResponse, error) {
	if req.Date == nil {
		return nil, status.Error(codes.InvalidArgument, "date is required")
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	c, err := h.deploy.RecordReinvestment(ctx, req.BucketId, amount, req.Date.AsTime())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &investorv1.RecordReinvestmentResponse{Contribution: contributionToProto(c)}, nil
}
