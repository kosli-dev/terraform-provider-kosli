#!/bin/bash

# Import an existing environment by name
terraform import kosli_environment.production_k8s production-k8s

# Import multiple environments
terraform import kosli_environment.staging_ecs staging-ecs
terraform import kosli_environment.data_lake data-lake-s3
