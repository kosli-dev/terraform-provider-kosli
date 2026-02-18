# Complete Logical Environment Example

This example demonstrates a comprehensive real-world scenario for managing logical environments in Kosli using Terraform. It shows how to aggregate physical environments into logical groups for unified visibility and reporting.

## Scenario

This example models a multi-region, multi-service production setup with staging environments:

### Physical Environments

**Production:**
- 2 Kubernetes clusters (US East, EU West)
- 1 ECS cluster (US East)
- 1 Lambda function group (US East)
- 1 S3 data lake

**Staging:**
- 1 Kubernetes cluster
- 1 ECS cluster

### Logical Environment Aggregations

The example creates several logical environments to group these physical environments in different ways:

1. **By Environment Tier:**
   - `production-all` - All production environments
   - `staging-all` - All staging environments

2. **By Geographic Region:**
   - `production-us-east` - All US East production environments
   - `production-eu-west` - All EU West production environments

3. **By Service Type:**
   - `production-kubernetes-all` - All Kubernetes clusters
   - `production-containers` - All container environments (K8S + ECS)
   - `production-serverless` - All serverless environments (Lambda + S3)

4. **Dynamic Configuration:**
   - `disaster-recovery-reference` - Uses data source to mirror production configuration

## Benefits

This approach provides several advantages:

- **Unified Visibility**: View compliance status across all production environments at once
- **Flexible Grouping**: Organize environments by region, service type, or tier
- **Simplified Reporting**: Generate compliance reports for logical groupings
- **Disaster Recovery**: Reference production configuration for DR planning
- **Team Organization**: Different teams can focus on specific logical environments

## Usage

1. Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in your Kosli credentials:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your values
   ```

2. Initialize Terraform:
   ```bash
   terraform init
   ```

3. Review the plan:
   ```bash
   terraform plan
   ```

4. Apply the configuration:
   ```bash
   terraform apply
   ```

5. View the outputs:
   ```bash
   terraform output
   ```

## Outputs

The example provides comprehensive outputs including:

- **physical_environments**: Map of all physical environments created
- **logical_environments**: Map of all logical environments and their aggregations
- **production_all_metadata**: Metadata from the production-all data source
- **environment_summary**: Summary statistics
- **all_production_environment_names**: Quick list of all production environment names
- **all_kubernetes_clusters**: List of all Kubernetes clusters
- **serverless_components**: List of all serverless components

## Real-World Adaptations

To adapt this example for your organization:

1. **Adjust Environment Names**: Replace with your actual environment names
2. **Add More Regions**: Extend with additional geographic regions as needed
3. **Custom Groupings**: Create logical environments that match your team structure
4. **Add Environment Types**: Include other environment types (docker, server) as needed
5. **Integration with CI/CD**: Reference these logical environments in your deployment pipelines

## Important Notes

- **Physical Environments Only**: Logical environments can ONLY contain physical environments (K8S, ECS, S3, docker, server, lambda), not other logical environments
- **API Validation**: The Kosli API validates that included environments exist and are physical
- **Drift Detection**: Changes to included_environments are automatically detected and managed by Terraform
- **Empty Lists**: Logical environments can have empty `included_environments` lists and be populated later

## Related Examples

- [Resource Example](../../resources/kosli_logical_environment/) - Basic resource usage
- [Data Source Example](../../data-sources/kosli_logical_environment/) - Querying logical environments
- [Environment Resource](../../resources/kosli_environment/) - Creating physical environments
