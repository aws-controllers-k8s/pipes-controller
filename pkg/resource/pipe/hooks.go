// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package pipe

import (
	"fmt"
	"reflect"
	"time"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	svcsdk "github.com/aws/aws-sdk-go/service/pipes"

	svcapitypes "github.com/aws-controllers-k8s/pipes-controller/apis/v1alpha1"
)

//  _____ _     _____ _      _____  ____  ____  _  ____  _____ _____
// /  __// \ |\/  __// \  /|/__ __\/  _ \/  __\/ \/  _ \/  __//  __/
// |  \  | | //|  \  | |\ ||  / \  | | //|  \/|| || | \|| |  _|  \
// |  /_ | \// |  /_ | | \||  | |  | |_\\|    /| || |_/|| |_//|  /_
// \____\\__/  \____\\_/  \|  \_/  \____/\_/\_\\_/\____/\____\\____\
//
// _  _    _  ____ ___  __  _   ____  _  ____  _____ ____
// \||/   / |/  _ \\  \//\||/  /  __\/ \/  __\/  __// ___\
//        | || / \| \  /       |  \/|| ||  \/||  \  |    \
//     /\_| || |-|| / /        |  __/| ||  __/|  /_ \___ |
//     \____/\_/ \|/_/         \_/   \_/\_/   \____\\____/

const (
	defaultRequeueDelay = time.Second * 5
)

var (
	requeueWaitWhileCreating = ackrequeue.NeededAfter(
		fmt.Errorf("pipe in status %q, requeueing", svcsdk.PipeStateCreating),
		defaultRequeueDelay,
	)

	requeueWaitWhileUpdating = ackrequeue.NeededAfter(
		fmt.Errorf("pipe in status %q, cannot be modified or deleted", svcsdk.PipeStateUpdating),
		defaultRequeueDelay,
	)

	requeueWaitWhileDeleting = ackrequeue.NeededAfter(
		fmt.Errorf("pipe in status %q, cannot be modified or deleted", svcsdk.PipeStateDeleting),
		defaultRequeueDelay,
	)
)

// hasNilDifference returns true if the supplied subjects' nilness is
// different
func hasNilDifference(a, b interface{}) bool {
	if isEmpty(a) || isEmpty(b) {
		if (isEmpty(a) && isNotEmpty(b)) || (isEmpty(b) && isNotEmpty(a)) {
			return true
		}
	}
	return false
}

// isEmpty checks the passed interface argument for Nil or empty struct value
// (with zero values). For interfaces, only 'i==nil' check is not sufficient.
// https://tour.golang.org/methods/12 More details:
// https://mangatmodi.medium.com/go-check-nil-interface-the-right-way-d142776edef1
func isEmpty(i interface{}) bool {
	if i == nil {
		return true
	}

	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		if reflect.ValueOf(i).IsNil() {
			return true
		}

		if reflect.ValueOf(i).Elem().Kind() == reflect.Struct {
			return reflect.ValueOf(i).Elem().IsZero()
		}
	}
	return false
}

func isNotEmpty(i interface{}) bool {
	return !isEmpty(i)
}

func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	aDesiredState := a.ko.Spec.DesiredState
	bDesiredState := b.ko.Spec.DesiredState

	// assumes API always returns desiredState
	running := aDesiredState == nil || *aDesiredState == "" || *aDesiredState == svcsdk.PipeStateRunning
	if !(running && *bDesiredState == svcsdk.PipeStateRunning) {
		if *aDesiredState != *bDesiredState {
			delta.Add("Spec.DesiredState", aDesiredState, bDesiredState)
		}
	}

	// hack (by the great @a-hilaly): forces a requeue in update if currentState != desiredState to reconcile
	// and update status fields in Kubernetes resource e.g., in case of UPDATE_FAILED
	if *bDesiredState != *b.ko.Status.CurrentState {
		// setting Spec. because Status. is not considered in delta logic
		delta.Add("Spec.CurrentState", *bDesiredState, *b.ko.Status.CurrentState)
	}

	if hasNilDifference(a.ko.Spec.SourceParameters, b.ko.Spec.SourceParameters) {
		delta.Add("Spec.SourceParameters", a.ko.Spec.SourceParameters, b.ko.Spec.SourceParameters)
	} else if a.ko.Spec.SourceParameters != nil && b.ko.Spec.SourceParameters != nil {
		if hasNilDifference(a.ko.Spec.SourceParameters.ActiveMQBrokerParameters, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters) {
			delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters)
		} else if a.ko.Spec.SourceParameters.ActiveMQBrokerParameters != nil && b.ko.Spec.SourceParameters.ActiveMQBrokerParameters != nil {
			if hasNilDifference(a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize) {
				delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize)
			} else if a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize != nil && b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize != nil {
				if *a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize != *b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize {
					delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.BatchSize)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials) {
				delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.Credentials", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials)
			} else if a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials != nil && b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials != nil {
				if hasNilDifference(a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth) {
					delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth)
				} else if a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth != nil && b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth != nil {
					if *a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth != *b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth {
						delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.Credentials.BasicAuth)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds) {
				delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds)
			} else if a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds != nil && b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds != nil {
				if *a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds != *b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds {
					delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.MaximumBatchingWindowInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName) {
				delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.QueueName", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName)
			} else if a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName != nil && b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName != nil {
				if *a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName != *b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName {
					delta.Add("Spec.SourceParameters.ActiveMQBrokerParameters.QueueName", a.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName, b.ko.Spec.SourceParameters.ActiveMQBrokerParameters.QueueName)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters, b.ko.Spec.SourceParameters.DynamoDBStreamParameters) {
			delta.Add("Spec.SourceParameters.DynamoDBStreamParameters", a.ko.Spec.SourceParameters.DynamoDBStreamParameters, b.ko.Spec.SourceParameters.DynamoDBStreamParameters)
		} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters != nil {
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.BatchSize", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize != nil {
				if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.BatchSize", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.BatchSize)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig != nil {
				if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN) {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN)
				} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN != nil {
					if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN {
						delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.DeadLetterConfig.ARN)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds != nil {
				if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumBatchingWindowInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds != nil {
				if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRecordAgeInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts != nil {
				if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.MaximumRetryAttempts)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure != nil {
				if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.OnPartialBatchItemFailure)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor != nil {
				if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.ParallelizationFactor)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition) {
				delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition)
			} else if a.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition != nil && b.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition != nil {
				if *a.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition != *b.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition {
					delta.Add("Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition", a.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition, b.ko.Spec.SourceParameters.DynamoDBStreamParameters.StartingPosition)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.SourceParameters.FilterCriteria, b.ko.Spec.SourceParameters.FilterCriteria) {
			delta.Add("Spec.SourceParameters.FilterCriteria", a.ko.Spec.SourceParameters.FilterCriteria, b.ko.Spec.SourceParameters.FilterCriteria)
		} else if a.ko.Spec.SourceParameters.FilterCriteria != nil && b.ko.Spec.SourceParameters.FilterCriteria != nil {
			if !reflect.DeepEqual(a.ko.Spec.SourceParameters.FilterCriteria.Filters, b.ko.Spec.SourceParameters.FilterCriteria.Filters) {
				delta.Add("Spec.SourceParameters.FilterCriteria.Filters", a.ko.Spec.SourceParameters.FilterCriteria.Filters, b.ko.Spec.SourceParameters.FilterCriteria.Filters)
			}
		}
		if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters, b.ko.Spec.SourceParameters.KinesisStreamParameters) {
			delta.Add("Spec.SourceParameters.KinesisStreamParameters", a.ko.Spec.SourceParameters.KinesisStreamParameters, b.ko.Spec.SourceParameters.KinesisStreamParameters)
		} else if a.ko.Spec.SourceParameters.KinesisStreamParameters != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters != nil {
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize, b.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.BatchSize", a.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize, b.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize != nil {
				if *a.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize != *b.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.BatchSize", a.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize, b.ko.Spec.SourceParameters.KinesisStreamParameters.BatchSize)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig, b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig", a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig, b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig != nil {
				if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN, b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN) {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN", a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN, b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN)
				} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN != nil {
					if *a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN != *b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN {
						delta.Add("Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN", a.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN, b.ko.Spec.SourceParameters.KinesisStreamParameters.DeadLetterConfig.ARN)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds != nil {
				if *a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds != *b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumBatchingWindowInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds", a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds != nil {
				if *a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds != *b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds", a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRecordAgeInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts", a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts != nil {
				if *a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts != *b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts", a.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts, b.ko.Spec.SourceParameters.KinesisStreamParameters.MaximumRetryAttempts)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure, b.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure", a.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure, b.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure != nil {
				if *a.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure != *b.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure", a.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure, b.ko.Spec.SourceParameters.KinesisStreamParameters.OnPartialBatchItemFailure)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor, b.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor", a.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor, b.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor != nil {
				if *a.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor != *b.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor", a.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor, b.ko.Spec.SourceParameters.KinesisStreamParameters.ParallelizationFactor)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition, b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.StartingPosition", a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition, b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition != nil {
				if *a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition != *b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.StartingPosition", a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition, b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPosition)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp, b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp) {
				delta.Add("Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp", a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp, b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp)
			} else if a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp != nil && b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp != nil {
				if !a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp.Equal(b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp) {
					delta.Add("Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp", a.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp, b.ko.Spec.SourceParameters.KinesisStreamParameters.StartingPositionTimestamp)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters) {
			delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters)
		} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters != nil {
			if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize) {
				delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize)
			} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize != nil {
				if *a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize != *b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize {
					delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.BatchSize)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID) {
				delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID)
			} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID != nil {
				if *a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID != *b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID {
					delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.ConsumerGroupID)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials) {
				delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials)
			} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials != nil {
				if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth) {
					delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth)
				} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth != nil {
					if *a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth != *b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth {
						delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.ClientCertificateTLSAuth)
					}
				}
				if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth) {
					delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth)
				} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth != nil {
					if *a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth != *b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth {
						delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.Credentials.SASLSCRAM512Auth)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds) {
				delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds)
			} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds != nil {
				if *a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds != *b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds {
					delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.MaximumBatchingWindowInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition) {
				delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition)
			} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition != nil {
				if *a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition != *b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition {
					delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.StartingPosition)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName) {
				delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName)
			} else if a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName != nil && b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName != nil {
				if *a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName != *b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName {
					delta.Add("Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName", a.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName, b.ko.Spec.SourceParameters.ManagedStreamingKafkaParameters.TopicName)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.SourceParameters.RabbitMQBrokerParameters, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters) {
			delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters)
		} else if a.ko.Spec.SourceParameters.RabbitMQBrokerParameters != nil && b.ko.Spec.SourceParameters.RabbitMQBrokerParameters != nil {
			if hasNilDifference(a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize) {
				delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize)
			} else if a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize != nil && b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize != nil {
				if *a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize != *b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize {
					delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.BatchSize)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials) {
				delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.Credentials", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials)
			} else if a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials != nil && b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials != nil {
				if hasNilDifference(a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth) {
					delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth)
				} else if a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth != nil && b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth != nil {
					if *a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth != *b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth {
						delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.Credentials.BasicAuth)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds) {
				delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds)
			} else if a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds != nil && b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds != nil {
				if *a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds != *b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds {
					delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.MaximumBatchingWindowInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName) {
				delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.QueueName", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName)
			} else if a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName != nil && b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName != nil {
				if *a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName != *b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName {
					delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.QueueName", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.QueueName)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost) {
				delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost)
			} else if a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost != nil && b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost != nil {
				if *a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost != *b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost {
					delta.Add("Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost", a.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost, b.ko.Spec.SourceParameters.RabbitMQBrokerParameters.VirtualHost)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters) {
			delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters)
		} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters != nil {
			if !ackcompare.SliceStringPEqual(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.AdditionalBootstrapServers, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.AdditionalBootstrapServers) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.AdditionalBootstrapServers", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.AdditionalBootstrapServers, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.AdditionalBootstrapServers)
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize != nil {
				if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.BatchSize)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID != nil {
				if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ConsumerGroupID)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials != nil {
				if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth) {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth)
				} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth != nil {
					if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth {
						delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.BasicAuth)
					}
				}
				if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth) {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth)
				} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth != nil {
					if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth {
						delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.ClientCertificateTLSAuth)
					}
				}
				if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth) {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth)
				} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth != nil {
					if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth {
						delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM256Auth)
					}
				}
				if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth) {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth)
				} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth != nil {
					if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth {
						delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.Credentials.SASLSCRAM512Auth)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds != nil {
				if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.MaximumBatchingWindowInSeconds)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate != nil {
				if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.ServerRootCaCertificate)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition != nil {
				if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.StartingPosition)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.TopicName", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName != nil {
				if *a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName != *b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.TopicName", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.TopicName)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC) {
				delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.VPC", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC)
			} else if a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC != nil && b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC != nil {
				if !ackcompare.SliceStringPEqual(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.SecurityGroup, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.SecurityGroup) {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.VPC.SecurityGroup", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.SecurityGroup, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.SecurityGroup)
				}
				if !ackcompare.SliceStringPEqual(a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.Subnets, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.Subnets) {
					delta.Add("Spec.SourceParameters.SelfManagedKafkaParameters.VPC.Subnets", a.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.Subnets, b.ko.Spec.SourceParameters.SelfManagedKafkaParameters.VPC.Subnets)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.SourceParameters.SQSQueueParameters, b.ko.Spec.SourceParameters.SQSQueueParameters) {
			delta.Add("Spec.SourceParameters.SQSQueueParameters", a.ko.Spec.SourceParameters.SQSQueueParameters, b.ko.Spec.SourceParameters.SQSQueueParameters)
		} else if a.ko.Spec.SourceParameters.SQSQueueParameters != nil && b.ko.Spec.SourceParameters.SQSQueueParameters != nil {
			if hasNilDifference(a.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize, b.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize) {
				delta.Add("Spec.SourceParameters.SQSQueueParameters.BatchSize", a.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize, b.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize)
			} else if a.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize != nil && b.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize != nil {
				if *a.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize != *b.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize {
					delta.Add("Spec.SourceParameters.SQSQueueParameters.BatchSize", a.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize, b.ko.Spec.SourceParameters.SQSQueueParameters.BatchSize)
				}
			}
			if hasNilDifference(a.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds) {
				delta.Add("Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds)
			} else if a.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds != nil && b.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds != nil {
				if *a.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds != *b.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds {
					delta.Add("Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds", a.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds, b.ko.Spec.SourceParameters.SQSQueueParameters.MaximumBatchingWindowInSeconds)
				}
			}
		}
	}
	if hasNilDifference(a.ko.Spec.TargetParameters, b.ko.Spec.TargetParameters) {
		delta.Add("Spec.TargetParameters", a.ko.Spec.TargetParameters, b.ko.Spec.TargetParameters)
	} else if a.ko.Spec.TargetParameters != nil && b.ko.Spec.TargetParameters != nil {
		if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters, b.ko.Spec.TargetParameters.BatchJobParameters) {
			delta.Add("Spec.TargetParameters.BatchJobParameters", a.ko.Spec.TargetParameters.BatchJobParameters, b.ko.Spec.TargetParameters.BatchJobParameters)
		} else if a.ko.Spec.TargetParameters.BatchJobParameters != nil && b.ko.Spec.TargetParameters.BatchJobParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties, b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties) {
				delta.Add("Spec.TargetParameters.BatchJobParameters.ArrayProperties", a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties, b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties)
			} else if a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties != nil && b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties != nil {
				if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size, b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size) {
					delta.Add("Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size", a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size, b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size)
				} else if a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size != nil && b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size != nil {
					if *a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size != *b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size {
						delta.Add("Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size", a.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size, b.ko.Spec.TargetParameters.BatchJobParameters.ArrayProperties.Size)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides) {
				delta.Add("Spec.TargetParameters.BatchJobParameters.ContainerOverrides", a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides)
			} else if a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides != nil && b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides != nil {
				if !ackcompare.SliceStringPEqual(a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Command, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Command) {
					delta.Add("Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Command", a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Command, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Command)
				}
				if !reflect.DeepEqual(a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Environment, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Environment) {
					delta.Add("Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Environment", a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Environment, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.Environment)
				}
				if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType) {
					delta.Add("Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType", a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType)
				} else if a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType != nil && b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType != nil {
					if *a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType != *b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType {
						delta.Add("Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType", a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.InstanceType)
					}
				}
				if !reflect.DeepEqual(a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.ResourceRequirements, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.ResourceRequirements) {
					delta.Add("Spec.TargetParameters.BatchJobParameters.ContainerOverrides.ResourceRequirements", a.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.ResourceRequirements, b.ko.Spec.TargetParameters.BatchJobParameters.ContainerOverrides.ResourceRequirements)
				}
			}
			if !reflect.DeepEqual(a.ko.Spec.TargetParameters.BatchJobParameters.DependsOn, b.ko.Spec.TargetParameters.BatchJobParameters.DependsOn) {
				delta.Add("Spec.TargetParameters.BatchJobParameters.DependsOn", a.ko.Spec.TargetParameters.BatchJobParameters.DependsOn, b.ko.Spec.TargetParameters.BatchJobParameters.DependsOn)
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition, b.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition) {
				delta.Add("Spec.TargetParameters.BatchJobParameters.JobDefinition", a.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition, b.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition)
			} else if a.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition != nil && b.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition != nil {
				if *a.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition != *b.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition {
					delta.Add("Spec.TargetParameters.BatchJobParameters.JobDefinition", a.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition, b.ko.Spec.TargetParameters.BatchJobParameters.JobDefinition)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.JobName, b.ko.Spec.TargetParameters.BatchJobParameters.JobName) {
				delta.Add("Spec.TargetParameters.BatchJobParameters.JobName", a.ko.Spec.TargetParameters.BatchJobParameters.JobName, b.ko.Spec.TargetParameters.BatchJobParameters.JobName)
			} else if a.ko.Spec.TargetParameters.BatchJobParameters.JobName != nil && b.ko.Spec.TargetParameters.BatchJobParameters.JobName != nil {
				if *a.ko.Spec.TargetParameters.BatchJobParameters.JobName != *b.ko.Spec.TargetParameters.BatchJobParameters.JobName {
					delta.Add("Spec.TargetParameters.BatchJobParameters.JobName", a.ko.Spec.TargetParameters.BatchJobParameters.JobName, b.ko.Spec.TargetParameters.BatchJobParameters.JobName)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.Parameters, b.ko.Spec.TargetParameters.BatchJobParameters.Parameters) {
				delta.Add("Spec.TargetParameters.BatchJobParameters.Parameters", a.ko.Spec.TargetParameters.BatchJobParameters.Parameters, b.ko.Spec.TargetParameters.BatchJobParameters.Parameters)
			} else if a.ko.Spec.TargetParameters.BatchJobParameters.Parameters != nil && b.ko.Spec.TargetParameters.BatchJobParameters.Parameters != nil {
				if !ackcompare.MapStringStringPEqual(a.ko.Spec.TargetParameters.BatchJobParameters.Parameters, b.ko.Spec.TargetParameters.BatchJobParameters.Parameters) {
					delta.Add("Spec.TargetParameters.BatchJobParameters.Parameters", a.ko.Spec.TargetParameters.BatchJobParameters.Parameters, b.ko.Spec.TargetParameters.BatchJobParameters.Parameters)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy, b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy) {
				delta.Add("Spec.TargetParameters.BatchJobParameters.RetryStrategy", a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy, b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy)
			} else if a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy != nil && b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy != nil {
				if hasNilDifference(a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts, b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts) {
					delta.Add("Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts", a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts, b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts)
				} else if a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts != nil && b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts != nil {
					if *a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts != *b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts {
						delta.Add("Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts", a.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts, b.ko.Spec.TargetParameters.BatchJobParameters.RetryStrategy.Attempts)
					}
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.CloudWatchLogsParameters, b.ko.Spec.TargetParameters.CloudWatchLogsParameters) {
			delta.Add("Spec.TargetParameters.CloudWatchLogsParameters", a.ko.Spec.TargetParameters.CloudWatchLogsParameters, b.ko.Spec.TargetParameters.CloudWatchLogsParameters)
		} else if a.ko.Spec.TargetParameters.CloudWatchLogsParameters != nil && b.ko.Spec.TargetParameters.CloudWatchLogsParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName, b.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName) {
				delta.Add("Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName", a.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName, b.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName)
			} else if a.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName != nil && b.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName != nil {
				if *a.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName != *b.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName {
					delta.Add("Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName", a.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName, b.ko.Spec.TargetParameters.CloudWatchLogsParameters.LogStreamName)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp, b.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp) {
				delta.Add("Spec.TargetParameters.CloudWatchLogsParameters.Timestamp", a.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp, b.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp)
			} else if a.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp != nil && b.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp != nil {
				if *a.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp != *b.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp {
					delta.Add("Spec.TargetParameters.CloudWatchLogsParameters.Timestamp", a.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp, b.ko.Spec.TargetParameters.CloudWatchLogsParameters.Timestamp)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters, b.ko.Spec.TargetParameters.ECSTaskParameters) {
			delta.Add("Spec.TargetParameters.ECSTaskParameters", a.ko.Spec.TargetParameters.ECSTaskParameters, b.ko.Spec.TargetParameters.ECSTaskParameters)
		} else if a.ko.Spec.TargetParameters.ECSTaskParameters != nil && b.ko.Spec.TargetParameters.ECSTaskParameters != nil {
			if !reflect.DeepEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.CapacityProviderStrategy, b.ko.Spec.TargetParameters.ECSTaskParameters.CapacityProviderStrategy) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.CapacityProviderStrategy", a.ko.Spec.TargetParameters.ECSTaskParameters.CapacityProviderStrategy, b.ko.Spec.TargetParameters.ECSTaskParameters.CapacityProviderStrategy)
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags, b.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags", a.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags, b.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags != *b.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags", a.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags, b.ko.Spec.TargetParameters.ECSTaskParameters.EnableECSManagedTags)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand, b.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand", a.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand, b.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand != *b.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand", a.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand, b.ko.Spec.TargetParameters.ECSTaskParameters.EnableExecuteCommand)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Group, b.ko.Spec.TargetParameters.ECSTaskParameters.Group) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.Group", a.ko.Spec.TargetParameters.ECSTaskParameters.Group, b.ko.Spec.TargetParameters.ECSTaskParameters.Group)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Group != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Group != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.Group != *b.ko.Spec.TargetParameters.ECSTaskParameters.Group {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Group", a.ko.Spec.TargetParameters.ECSTaskParameters.Group, b.ko.Spec.TargetParameters.ECSTaskParameters.Group)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType, b.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.LaunchType", a.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType, b.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType != *b.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.LaunchType", a.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType, b.ko.Spec.TargetParameters.ECSTaskParameters.LaunchType)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration", a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration != nil {
				if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration", a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration)
				} else if a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration != nil {
					if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP) {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP", a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP)
					} else if a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP != nil {
						if *a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP != *b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP {
							delta.Add("Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP", a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.AssignPublicIP)
						}
					}
					if !ackcompare.SliceStringPEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.SecurityGroups, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.SecurityGroups) {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.SecurityGroups", a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.SecurityGroups, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.SecurityGroups)
					}
					if !ackcompare.SliceStringPEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.Subnets, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.Subnets) {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.Subnets", a.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.Subnets, b.ko.Spec.TargetParameters.ECSTaskParameters.NetworkConfiguration.AWSVPCConfiguration.Subnets)
					}
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides != nil {
				if !reflect.DeepEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ContainerOverrides, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ContainerOverrides) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.ContainerOverrides", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ContainerOverrides, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ContainerOverrides)
				}
				if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.CPU", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU)
				} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU != nil {
					if *a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU != *b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.CPU", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.CPU)
					}
				}
				if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage)
				} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage != nil {
					if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB) {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB)
					} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB != nil {
						if *a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB != *b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB {
							delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.EphemeralStorage.SizeInGiB)
						}
					}
				}
				if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN)
				} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN != nil {
					if *a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN != *b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.ExecutionRoleARN)
					}
				}
				if !reflect.DeepEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.InferenceAcceleratorOverrides, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.InferenceAcceleratorOverrides) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.InferenceAcceleratorOverrides", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.InferenceAcceleratorOverrides, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.InferenceAcceleratorOverrides)
				}
				if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.Memory", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory)
				} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory != nil {
					if *a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory != *b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.Memory", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.Memory)
					}
				}
				if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN) {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN)
				} else if a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN != nil {
					if *a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN != *b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN {
						delta.Add("Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN", a.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN, b.ko.Spec.TargetParameters.ECSTaskParameters.Overrides.TaskRoleARN)
					}
				}
			}
			if !reflect.DeepEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.PlacementConstraints, b.ko.Spec.TargetParameters.ECSTaskParameters.PlacementConstraints) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.PlacementConstraints", a.ko.Spec.TargetParameters.ECSTaskParameters.PlacementConstraints, b.ko.Spec.TargetParameters.ECSTaskParameters.PlacementConstraints)
			}
			if !reflect.DeepEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.PlacementStrategy, b.ko.Spec.TargetParameters.ECSTaskParameters.PlacementStrategy) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.PlacementStrategy", a.ko.Spec.TargetParameters.ECSTaskParameters.PlacementStrategy, b.ko.Spec.TargetParameters.ECSTaskParameters.PlacementStrategy)
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion, b.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.PlatformVersion", a.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion, b.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion != *b.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.PlatformVersion", a.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion, b.ko.Spec.TargetParameters.ECSTaskParameters.PlatformVersion)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags, b.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.PropagateTags", a.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags, b.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags != *b.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.PropagateTags", a.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags, b.ko.Spec.TargetParameters.ECSTaskParameters.PropagateTags)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID, b.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.ReferenceID", a.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID, b.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID != *b.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.ReferenceID", a.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID, b.ko.Spec.TargetParameters.ECSTaskParameters.ReferenceID)
				}
			}
			if !reflect.DeepEqual(a.ko.Spec.TargetParameters.ECSTaskParameters.Tags, b.ko.Spec.TargetParameters.ECSTaskParameters.Tags) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.Tags", a.ko.Spec.TargetParameters.ECSTaskParameters.Tags, b.ko.Spec.TargetParameters.ECSTaskParameters.Tags)
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount, b.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.TaskCount", a.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount, b.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount != *b.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.TaskCount", a.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount, b.ko.Spec.TargetParameters.ECSTaskParameters.TaskCount)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN, b.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN) {
				delta.Add("Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN", a.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN, b.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN)
			} else if a.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN != nil && b.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN != nil {
				if *a.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN != *b.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN {
					delta.Add("Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN", a.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN, b.ko.Spec.TargetParameters.ECSTaskParameters.TaskDefinitionARN)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.EventBridgeEventBusParameters, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters) {
			delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters)
		} else if a.ko.Spec.TargetParameters.EventBridgeEventBusParameters != nil && b.ko.Spec.TargetParameters.EventBridgeEventBusParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType) {
				delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.DetailType", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType)
			} else if a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType != nil && b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType != nil {
				if *a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType != *b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType {
					delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.DetailType", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.DetailType)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID) {
				delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID)
			} else if a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID != nil && b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID != nil {
				if *a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID != *b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID {
					delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.EndpointID)
				}
			}
			if !ackcompare.SliceStringPEqual(a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Resources, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Resources) {
				delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.Resources", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Resources, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Resources)
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source) {
				delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.Source", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source)
			} else if a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source != nil && b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source != nil {
				if *a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source != *b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source {
					delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.Source", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Source)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time) {
				delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.Time", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time)
			} else if a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time != nil && b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time != nil {
				if *a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time != *b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time {
					delta.Add("Spec.TargetParameters.EventBridgeEventBusParameters.Time", a.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time, b.ko.Spec.TargetParameters.EventBridgeEventBusParameters.Time)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.HTTPParameters, b.ko.Spec.TargetParameters.HTTPParameters) {
			delta.Add("Spec.TargetParameters.HTTPParameters", a.ko.Spec.TargetParameters.HTTPParameters, b.ko.Spec.TargetParameters.HTTPParameters)
		} else if a.ko.Spec.TargetParameters.HTTPParameters != nil && b.ko.Spec.TargetParameters.HTTPParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters, b.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters) {
				delta.Add("Spec.TargetParameters.HTTPParameters.HeaderParameters", a.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters, b.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters)
			} else if a.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters != nil && b.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters != nil {
				if !ackcompare.MapStringStringPEqual(a.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters, b.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters) {
					delta.Add("Spec.TargetParameters.HTTPParameters.HeaderParameters", a.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters, b.ko.Spec.TargetParameters.HTTPParameters.HeaderParameters)
				}
			}
			if !ackcompare.SliceStringPEqual(a.ko.Spec.TargetParameters.HTTPParameters.PathParameterValues, b.ko.Spec.TargetParameters.HTTPParameters.PathParameterValues) {
				delta.Add("Spec.TargetParameters.HTTPParameters.PathParameterValues", a.ko.Spec.TargetParameters.HTTPParameters.PathParameterValues, b.ko.Spec.TargetParameters.HTTPParameters.PathParameterValues)
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters) {
				delta.Add("Spec.TargetParameters.HTTPParameters.QueryStringParameters", a.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters)
			} else if a.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters != nil && b.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters != nil {
				if !ackcompare.MapStringStringPEqual(a.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters) {
					delta.Add("Spec.TargetParameters.HTTPParameters.QueryStringParameters", a.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.TargetParameters.HTTPParameters.QueryStringParameters)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.InputTemplate, b.ko.Spec.TargetParameters.InputTemplate) {
			delta.Add("Spec.TargetParameters.InputTemplate", a.ko.Spec.TargetParameters.InputTemplate, b.ko.Spec.TargetParameters.InputTemplate)
		} else if a.ko.Spec.TargetParameters.InputTemplate != nil && b.ko.Spec.TargetParameters.InputTemplate != nil {
			if *a.ko.Spec.TargetParameters.InputTemplate != *b.ko.Spec.TargetParameters.InputTemplate {
				delta.Add("Spec.TargetParameters.InputTemplate", a.ko.Spec.TargetParameters.InputTemplate, b.ko.Spec.TargetParameters.InputTemplate)
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.KinesisStreamParameters, b.ko.Spec.TargetParameters.KinesisStreamParameters) {
			delta.Add("Spec.TargetParameters.KinesisStreamParameters", a.ko.Spec.TargetParameters.KinesisStreamParameters, b.ko.Spec.TargetParameters.KinesisStreamParameters)
		} else if a.ko.Spec.TargetParameters.KinesisStreamParameters != nil && b.ko.Spec.TargetParameters.KinesisStreamParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey, b.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey) {
				delta.Add("Spec.TargetParameters.KinesisStreamParameters.PartitionKey", a.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey, b.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey)
			} else if a.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey != nil && b.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey != nil {
				if *a.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey != *b.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey {
					delta.Add("Spec.TargetParameters.KinesisStreamParameters.PartitionKey", a.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey, b.ko.Spec.TargetParameters.KinesisStreamParameters.PartitionKey)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.LambdaFunctionParameters, b.ko.Spec.TargetParameters.LambdaFunctionParameters) {
			delta.Add("Spec.TargetParameters.LambdaFunctionParameters", a.ko.Spec.TargetParameters.LambdaFunctionParameters, b.ko.Spec.TargetParameters.LambdaFunctionParameters)
		} else if a.ko.Spec.TargetParameters.LambdaFunctionParameters != nil && b.ko.Spec.TargetParameters.LambdaFunctionParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType, b.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType) {
				delta.Add("Spec.TargetParameters.LambdaFunctionParameters.InvocationType", a.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType, b.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType)
			} else if a.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType != nil && b.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType != nil {
				if *a.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType != *b.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType {
					delta.Add("Spec.TargetParameters.LambdaFunctionParameters.InvocationType", a.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType, b.ko.Spec.TargetParameters.LambdaFunctionParameters.InvocationType)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.RedshiftDataParameters, b.ko.Spec.TargetParameters.RedshiftDataParameters) {
			delta.Add("Spec.TargetParameters.RedshiftDataParameters", a.ko.Spec.TargetParameters.RedshiftDataParameters, b.ko.Spec.TargetParameters.RedshiftDataParameters)
		} else if a.ko.Spec.TargetParameters.RedshiftDataParameters != nil && b.ko.Spec.TargetParameters.RedshiftDataParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.RedshiftDataParameters.Database, b.ko.Spec.TargetParameters.RedshiftDataParameters.Database) {
				delta.Add("Spec.TargetParameters.RedshiftDataParameters.Database", a.ko.Spec.TargetParameters.RedshiftDataParameters.Database, b.ko.Spec.TargetParameters.RedshiftDataParameters.Database)
			} else if a.ko.Spec.TargetParameters.RedshiftDataParameters.Database != nil && b.ko.Spec.TargetParameters.RedshiftDataParameters.Database != nil {
				if *a.ko.Spec.TargetParameters.RedshiftDataParameters.Database != *b.ko.Spec.TargetParameters.RedshiftDataParameters.Database {
					delta.Add("Spec.TargetParameters.RedshiftDataParameters.Database", a.ko.Spec.TargetParameters.RedshiftDataParameters.Database, b.ko.Spec.TargetParameters.RedshiftDataParameters.Database)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser, b.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser) {
				delta.Add("Spec.TargetParameters.RedshiftDataParameters.DBUser", a.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser, b.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser)
			} else if a.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser != nil && b.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser != nil {
				if *a.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser != *b.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser {
					delta.Add("Spec.TargetParameters.RedshiftDataParameters.DBUser", a.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser, b.ko.Spec.TargetParameters.RedshiftDataParameters.DBUser)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN, b.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN) {
				delta.Add("Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN", a.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN, b.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN)
			} else if a.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN != nil && b.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN != nil {
				if *a.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN != *b.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN {
					delta.Add("Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN", a.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN, b.ko.Spec.TargetParameters.RedshiftDataParameters.SecretManagerARN)
				}
			}
			if !ackcompare.SliceStringPEqual(a.ko.Spec.TargetParameters.RedshiftDataParameters.SQLs, b.ko.Spec.TargetParameters.RedshiftDataParameters.SQLs) {
				delta.Add("Spec.TargetParameters.RedshiftDataParameters.SQLs", a.ko.Spec.TargetParameters.RedshiftDataParameters.SQLs, b.ko.Spec.TargetParameters.RedshiftDataParameters.SQLs)
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName, b.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName) {
				delta.Add("Spec.TargetParameters.RedshiftDataParameters.StatementName", a.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName, b.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName)
			} else if a.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName != nil && b.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName != nil {
				if *a.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName != *b.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName {
					delta.Add("Spec.TargetParameters.RedshiftDataParameters.StatementName", a.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName, b.ko.Spec.TargetParameters.RedshiftDataParameters.StatementName)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent, b.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent) {
				delta.Add("Spec.TargetParameters.RedshiftDataParameters.WithEvent", a.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent, b.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent)
			} else if a.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent != nil && b.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent != nil {
				if *a.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent != *b.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent {
					delta.Add("Spec.TargetParameters.RedshiftDataParameters.WithEvent", a.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent, b.ko.Spec.TargetParameters.RedshiftDataParameters.WithEvent)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.SageMakerPipelineParameters, b.ko.Spec.TargetParameters.SageMakerPipelineParameters) {
			delta.Add("Spec.TargetParameters.SageMakerPipelineParameters", a.ko.Spec.TargetParameters.SageMakerPipelineParameters, b.ko.Spec.TargetParameters.SageMakerPipelineParameters)
		} else if a.ko.Spec.TargetParameters.SageMakerPipelineParameters != nil && b.ko.Spec.TargetParameters.SageMakerPipelineParameters != nil {
			if !reflect.DeepEqual(a.ko.Spec.TargetParameters.SageMakerPipelineParameters.PipelineParameterList, b.ko.Spec.TargetParameters.SageMakerPipelineParameters.PipelineParameterList) {
				delta.Add("Spec.TargetParameters.SageMakerPipelineParameters.PipelineParameterList", a.ko.Spec.TargetParameters.SageMakerPipelineParameters.PipelineParameterList, b.ko.Spec.TargetParameters.SageMakerPipelineParameters.PipelineParameterList)
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.SQSQueueParameters, b.ko.Spec.TargetParameters.SQSQueueParameters) {
			delta.Add("Spec.TargetParameters.SQSQueueParameters", a.ko.Spec.TargetParameters.SQSQueueParameters, b.ko.Spec.TargetParameters.SQSQueueParameters)
		} else if a.ko.Spec.TargetParameters.SQSQueueParameters != nil && b.ko.Spec.TargetParameters.SQSQueueParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID, b.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID) {
				delta.Add("Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID", a.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID, b.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID)
			} else if a.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID != nil && b.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID != nil {
				if *a.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID != *b.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID {
					delta.Add("Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID", a.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID, b.ko.Spec.TargetParameters.SQSQueueParameters.MessageDeduplicationID)
				}
			}
			if hasNilDifference(a.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID, b.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID) {
				delta.Add("Spec.TargetParameters.SQSQueueParameters.MessageGroupID", a.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID, b.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID)
			} else if a.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID != nil && b.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID != nil {
				if *a.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID != *b.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID {
					delta.Add("Spec.TargetParameters.SQSQueueParameters.MessageGroupID", a.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID, b.ko.Spec.TargetParameters.SQSQueueParameters.MessageGroupID)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters, b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters) {
			delta.Add("Spec.TargetParameters.StepFunctionStateMachineParameters", a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters, b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters)
		} else if a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters != nil && b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters != nil {
			if hasNilDifference(a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType, b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType) {
				delta.Add("Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType", a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType, b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType)
			} else if a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType != nil && b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType != nil {
				if *a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType != *b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType {
					delta.Add("Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType", a.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType, b.ko.Spec.TargetParameters.StepFunctionStateMachineParameters.InvocationType)
				}
			}
		}
	}

	if hasNilDifference(a.ko.Spec.EnrichmentParameters, b.ko.Spec.EnrichmentParameters) {
		delta.Add("Spec.EnrichmentParameters", a.ko.Spec.EnrichmentParameters, b.ko.Spec.EnrichmentParameters)
	} else if a.ko.Spec.EnrichmentParameters != nil && b.ko.Spec.EnrichmentParameters != nil {
		if hasNilDifference(a.ko.Spec.EnrichmentParameters.HTTPParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters) {
			delta.Add("Spec.EnrichmentParameters.HTTPParameters", a.ko.Spec.EnrichmentParameters.HTTPParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters)
		} else if a.ko.Spec.EnrichmentParameters.HTTPParameters != nil && b.ko.Spec.EnrichmentParameters.HTTPParameters != nil {
			if hasNilDifference(a.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters) {
				delta.Add("Spec.EnrichmentParameters.HTTPParameters.HeaderParameters", a.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters)
			} else if a.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters != nil && b.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters != nil {
				if !ackcompare.MapStringStringPEqual(a.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters) {
					delta.Add("Spec.EnrichmentParameters.HTTPParameters.HeaderParameters", a.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.HeaderParameters)
				}
			}
			if !ackcompare.SliceStringPEqual(a.ko.Spec.EnrichmentParameters.HTTPParameters.PathParameterValues, b.ko.Spec.EnrichmentParameters.HTTPParameters.PathParameterValues) {
				delta.Add("Spec.EnrichmentParameters.HTTPParameters.PathParameterValues", a.ko.Spec.EnrichmentParameters.HTTPParameters.PathParameterValues, b.ko.Spec.EnrichmentParameters.HTTPParameters.PathParameterValues)
			}
			if hasNilDifference(a.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters) {
				delta.Add("Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters", a.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters)
			} else if a.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters != nil && b.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters != nil {
				if !ackcompare.MapStringStringPEqual(a.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters) {
					delta.Add("Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters", a.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters, b.ko.Spec.EnrichmentParameters.HTTPParameters.QueryStringParameters)
				}
			}
		}
		if hasNilDifference(a.ko.Spec.EnrichmentParameters.InputTemplate, b.ko.Spec.EnrichmentParameters.InputTemplate) {
			delta.Add("Spec.EnrichmentParameters.InputTemplate", a.ko.Spec.EnrichmentParameters.InputTemplate, b.ko.Spec.EnrichmentParameters.InputTemplate)
		} else if a.ko.Spec.EnrichmentParameters.InputTemplate != nil && b.ko.Spec.EnrichmentParameters.InputTemplate != nil {
			if *a.ko.Spec.EnrichmentParameters.InputTemplate != *b.ko.Spec.EnrichmentParameters.InputTemplate {
				delta.Add("Spec.EnrichmentParameters.InputTemplate", a.ko.Spec.EnrichmentParameters.InputTemplate, b.ko.Spec.EnrichmentParameters.InputTemplate)
			}
		}
	}
}

// pipeAvailable returns true if the supplied Pipe is in a running status
func pipeAvailable(r *resource) bool {
	if r.ko.Status.CurrentState == nil {
		return false
	}
	state := *r.ko.Status.CurrentState
	return state == svcsdk.PipeStateRunning
}

// pipeInMutatingState returns true if the supplied Pipe is in the process of
// being modified
func pipeInMutatingState(r *resource) bool {
	if r.ko.Status.CurrentState == nil {
		return false
	}
	state := *r.ko.Status.CurrentState

	mutatingStates := []string{
		svcsdk.PipeStateCreating,
		svcsdk.PipeStateStarting,
		svcsdk.PipeStateStopping,
		svcsdk.PipeStateUpdating,
		svcsdk.PipeStateDeleting,
	}

	for _, s := range mutatingStates {
		if state == s {
			return true
		}
	}
	return false
}

// if an optional desired field value is nil explicitly unset it in the request
// input
func unsetRemovedSpecFields(
	delta *ackcompare.Delta,
	spec svcapitypes.PipeSpec,
	input *svcsdk.UpdatePipeInput,
) {
	if delta.DifferentAt("Spec.Description") {
		if spec.Description == nil {
			input.SetDescription("")
		}
	}

	if delta.DifferentAt("Spec.DesiredState") {
		if spec.DesiredState == nil {
			input.SetDesiredState("")
		}
	}
}
