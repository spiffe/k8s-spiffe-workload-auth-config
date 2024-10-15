# K8s SPIFFE Workload Auth Config

[![Apache 2.0 License](https://img.shields.io/github/license/spiffe/helm-charts)](https://opensource.org/licenses/Apache-2.0)
[![Development Phase](https://github.com/spiffe/spiffe/blob/main/.img/maturity/dev.svg)](https://github.com/spiffe/spiffe/blob/main/MATURITY.md#development)

A tool to help manage the Kubernetes AuthenticationConfiguration file by injecting the SPIFFE trust bundle

## How it Works

### Details

In order to establish a trust relationship between Kubernetes and SPIRE for JWT authentication, Kubernetes needs to be configured to point at the spiffe-oidc-discovery-provider. This communication needs to be secured via a TLS certificate. When using SPIFFE itself to secure the discovery provider, Kubernetes needs to be made aware of the SPIFFE Trust Bundle. This is achieved via this tool by fetching the issuer and trust bundle from SPIFFE and updating the certificateAuthority property of the AuthenticationConfiguration file as needed.

### Building

```
go build .
```

### Usage

```
./k8s-spiffe-workload-auth-config <sourcefile> dest.yaml
```

### Environment Variables

* SPIFFE_TRUST_DOMAIN
* SPIFFE_ENDPOINT_SOCKET

### Configure Kubernetes

Add --authentication-config=/etc/kubernetes/pki/auth-config.yaml to the kube-apiserver command line.

### Configure SPIRE entries

Example configuration for this service:
```
spire-server entry create \
  -socketPath /var/run/spire/server/sockets/main/private/api.sock \
  -parentID spiffe://example.com/node/node1 \
  -spiffeID spiffe://example.com/workload/k8s-spiffe-workload-auth-config \
  -selector systemd:id:k8s-spiffe-workload-auth-config.service
```

Example discovery provider:
```
spire-server entry create \
  -socketPath /var/run/spire/server/sockets/main/private/api.sock \
  -parentID spiffe://example.com/node/node1 \
  -spiffeID spiffe://example.com/workload/oidc-discovery-provider \
  -selector systemd:id:k8s-spiffe-oidc-discovery-provider.service \
  -dns oidc-discovery-provider.example.com
```

### Kubernetes Clients

Use in conjunction with https://github.com/spiffe/k8s-spiffe-workload-jwt-exec-auth for Kubernetes client authentication via SPIFFE to the Kubernetes cluster.
