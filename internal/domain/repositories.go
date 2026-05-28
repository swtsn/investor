package domain

import "context"

type BucketRepository interface {
	ListBuckets(ctx context.Context) ([]Bucket, error)
	GetBucket(ctx context.Context, id int64) (Bucket, error)
	CreateBucket(ctx context.Context, b Bucket) (Bucket, error)
	UpdateBucket(ctx context.Context, b Bucket) error
	ListAllocations(ctx context.Context, bucketID int64) ([]Allocation, error)
	UpsertAllocation(ctx context.Context, a Allocation) error
	DeleteAllocation(ctx context.Context, id int64) error
}

type ContributionRepository interface {
	CreateContribution(ctx context.Context, c Contribution) (Contribution, error)
	ListByBucket(ctx context.Context, bucketID int64) ([]Contribution, error)
	ListByBucketAndMonth(ctx context.Context, bucketID int64, year, month int) ([]Contribution, error)
}

type DeploymentRepository interface {
	// CreateDeployment writes a Deployment and its sources. Caller must wrap in Store.InTx.
	CreateDeployment(ctx context.Context, d Deployment, sources []DeploymentSource) (Deployment, error)
	ListByBucket(ctx context.Context, bucketID int64) ([]Deployment, error)
	ListByBucketAndMonth(ctx context.Context, bucketID int64, year, month int) ([]Deployment, error)
	ListSourcesByDeployment(ctx context.Context, deploymentID int64) ([]DeploymentSource, error)
}

type BudgetEventRepository interface {
	CreateBudgetEvent(ctx context.Context, e BudgetEvent) (BudgetEvent, error)
	ListByMonth(ctx context.Context, year, month int) ([]BudgetEvent, error)
}
