# ecr_reverse_proxy

Reverse Proxy for ECR. It is useful in scenarios where the client doesn't have credentials (e.g. Kubernetes running on-prem)

This should be run on an EC2 instance that is assuming an IAM role with permission to the ECR repository.

Inspired by https://github.com/marjamis/ecr_reverse_proxy