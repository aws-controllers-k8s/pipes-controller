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

"""Helper functions for EventBridgePipes e2e tests
"""

import logging

class PipesValidator:
    def __init__(self, pipes_client):
        self.pipes_client = pipes_client

    def get_pipe(self, pipe_name: str) -> dict:
        try:
            resp = self.pipes_client.describe_pipe(
                Name=pipe_name
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def pipe_exists(self, pipe_name) -> bool:
        return self.get_pipe(pipe_name) is not None

    def get_resource_tags(self, resource_arn: str):
        resource_tags = self.pipes_client.list_tags_for_resource(
            resourceArn=resource_arn,
        )
        return resource_tags['tags']