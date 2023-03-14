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
	"context"

	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/pipes"
)

// updatePipeTags uses TagResource and UntagResource to add, remove and update
// a pipe tags.
func (rm *resourceManager) updatePipeTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updatePipeTags")
	defer exit(err)

	addedOrUpdated, removed := compareMaps(latest.ko.Spec.Tags, desired.ko.Spec.Tags)

	if len(removed) > 0 {
		input := &svcsdk.UntagResourceInput{
			ResourceArn: (*string)(desired.ko.Status.ACKResourceMetadata.ARN),
			TagKeys:     removed,
		}
		_, err = rm.sdkapi.UntagResourceWithContext(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "UntagResource", err)
		if err != nil {
			return err
		}
	}

	if len(addedOrUpdated) > 0 {
		input := &svcsdk.TagResourceInput{
			ResourceArn: (*string)(desired.ko.Status.ACKResourceMetadata.ARN),
			Tags:        addedOrUpdated,
		}
		_, err = rm.sdkapi.TagResourceWithContext(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "TagResource", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// compareMaps compares two string to string maps and returns three outputs: a
// map of the new key/values observed, a list of the keys of the removed values
// and a map containing the updated keys and their new values.
func compareMaps(
	a map[string]*string,
	b map[string]*string,
) (addedOrUpdated map[string]*string, removed []*string) {
	addedOrUpdated = make(map[string]*string)
	visited := make(map[string]bool, len(a))
	for keyA, valueA := range a {
		valueB, found := b[keyA]
		if !found {
			removed = append(removed, &keyA)
			continue
		}
		if *valueA != *valueB {
			addedOrUpdated[keyA] = valueB
		}
		visited[keyA] = true
	}
	for keyB, valueB := range b {
		_, found := a[keyB]
		if !found {
			addedOrUpdated[keyB] = valueB
		}
	}
	return
}
