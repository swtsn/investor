package domain

import "errors"

var ErrNotFound = errors.New("not found")
var ErrBucketIsFlat = errors.New("bucket is flat: allocations not supported")
var ErrAllocationHasDeployments = errors.New("allocation has existing deployments")
