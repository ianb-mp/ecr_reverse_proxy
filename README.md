# ecr_reverse_proxy

A reverse proxy for ECR. It is useful in scenarios where the client doesn't have credentials (e.g. Kubernetes running on-prem). Those clients would point at this proxy.

This should be run on an EC2 instance that is assuming an IAM role with permission to the ECR repositories being used.

**WARNING** the proxy is insecure by default: no TLS, no IP restrictions. Only use over a VPN. Use at your own risk!

## Usage

```
./ecr_reverse_proxy \
  -ecr_registry <account_id>.dkr.ecr.<region>.amazonaws.com \
  -proxy_hostname myproxyhost
```

## Credit

Inspired by https://github.com/marjamis/ecr_reverse_proxy