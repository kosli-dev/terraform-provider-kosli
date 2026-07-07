# Import an existing API key using the "<service_account_name>/<key_id>" format.
# Note: the raw key value cannot be recovered on import (it is only returned once
# at creation), so the "key" attribute will be empty after import.
terraform import kosli_service_account_api_key.ci_key ci-pipeline/01HXYZ0123456789ABCDEFGHIJ
