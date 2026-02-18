#!/bin/bash

# Import an existing logical environment by name
terraform import kosli_logical_environment.production_all production-aggregate

# Import multiple logical environments
terraform import kosli_logical_environment.cloud_services cloud-services
terraform import kosli_logical_environment.future_environments future-environments
