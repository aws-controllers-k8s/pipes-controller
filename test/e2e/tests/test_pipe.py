# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the EventBridgePipes Pipe resource
"""

import logging
import time
from typing import Dict

import pytest

from acktest import tags
from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_pipes_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import PipesValidator

RESOURCE_PLURAL = "pipes"

CREATE_WAIT_AFTER_SECONDS = 45
UPDATE_WAIT_AFTER_SECONDS = 75
DELETE_WAIT_AFTER_SECONDS = 60

@pytest.fixture(scope="module")
def simple_pipe():
    resource_name = random_suffix_name("ack-test-pipe", 24)

    resources = get_bootstrap_resources()
    replacements = REPLACEMENT_VALUES.copy()
    replacements["PIPE_NAME"] = resource_name
    replacements["PIPE_ROLE_ARN"] = resources.PipeRole.arn
    replacements["SOURCE_ARN"] = resources.SourceQueue.arn
    replacements["TARGET_ARN"] = resources.TargetQueue.arn

    resource_data = load_pipes_resource(
        "pipe",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)

    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Get latest pipe CR
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 15, 15)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestPipe:
    def test_create_delete_with_tags(self, pipes_client, simple_pipe):
        (ref, cr) = simple_pipe

        pipe_name = cr["spec"]["name"]
        pipe_arn = cr["status"]["ackResourceMetadata"]["arn"]

        pipes_validator = PipesValidator(pipes_client)
        # verify that pipe exists
        assert pipes_validator.pipe_exists(pipe_name)

        # verify that pipe tags are created
        pipe_tags = pipes_validator.get_resource_tags(pipe_arn)
        tags.assert_ack_system_tags(
            tags=pipe_tags,
        )
        tags.assert_equal_without_ack_tags(
            expected=cr["spec"]["tags"],
            actual=pipe_tags,
        )

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 15, 15)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check pipe doesn't exist
        assert not pipes_validator.pipe_exists(pipe_name)

    def test_simple_update(self, pipes_client, simple_pipe):
        (ref, cr) = simple_pipe

        pipe_name = cr["spec"]["name"]
        pipe_arn = cr["status"]["ackResourceMetadata"]["arn"]

        pipes_validator = PipesValidator(pipes_client)
        # verify that pipe exists
        assert pipes_validator.pipe_exists(pipe_name)


        # New spec fields
        cr["spec"]["tags"] = {
            "env": "prod",
        }
        cr["spec"]["description"] = "testing pipe created ACK - updated"
        
        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # verify that pipe description and tags are updated
        pipe = pipes_validator.get_pipe(pipe_name)
        assert pipe["Description"] == "testing pipe created ACK - updated"

        pipe_tags = pipes_validator.get_resource_tags(pipe_arn)
        tags.assert_ack_system_tags(
            tags=pipe_tags,
        )
        tags.assert_equal_without_ack_tags(
            expected=cr["spec"]["tags"],
            actual=pipe_tags,
        )

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 15, 15)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check pipe doesn't exist
        assert not pipes_validator.pipe_exists(pipe_name)

    def test_pipe_update_state(self, pipes_client, simple_pipe):
        (ref, cr) = simple_pipe

        pipe_name = cr["spec"]["name"]

        pipes_validator = PipesValidator(pipes_client)
        # verify that pipe exists
        assert pipes_validator.pipe_exists(pipe_name)

        cr["spec"]["desiredState"] = "STOPPED"
        
        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        pipe = pipes_validator.get_pipe(pipe_name)
        assert pipe["DesiredState"] == "STOPPED"
        assert pipe["CurrentState"] == "STOPPED"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 15, 15)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check pipe doesn't exist
        assert not pipes_validator.pipe_exists(pipe_name)