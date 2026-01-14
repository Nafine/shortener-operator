# shortener-operator
The UrlShortener operator can be used for deploying and managing 
[UrlShortener](https://github.com/Nafine/url-shortener) application into kubernetes.

The architecture of the Url Shortener Operator follows the basic controller pattern: the Operator container with the 
controller is deployed into a Pod and listens for incoming resources with Kind: UrlShortener.

## Getting Started

### Prerequisites
- go version v1.24.6+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Operator Installation

Using install.yaml:

```sh
kubectl apply -f https://raw.githubusercontent.com/nafine/shortener-operator/main/dist/install.yaml
```

Or you can use provided Helm Chart located in `dist/chart`.

### Configuration

```yaml
apiVersion: shortener.nafine.dev/v1alpha1
kind: UrlShortener
metadata:
  labels:
    app.kubernetes.io/name: shortener-operator
    app.kubernetes.io/managed-by: kustomize
  name: urlshortener-sample
spec:
  replicas: 3
  appEnv: "local"
  storageDsnSecretRef:
      name: storage-dsn
      key: dsn
  apiKeysSecretRef:
      name: keys-secret
      key: apiKeys.yaml
  http:
    port: 8080
    timeout: 4s
    idleTimeout: 10s
```

> [!NOTE]
> spec..http.port defines port on which ==container and service== will listen to.

#### Storage

For application to work you need to deploy an instance of PostgreSQL database.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: shortener-db
spec:
  selector:
    app: shortener-db
  ports:
    - protocol: TCP
      port: 5432
      targetPort: 5432
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shortener-db
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: shortener-db
  template:
    metadata:
      labels:
        app: shortener-db
    spec:
      containers:
        - name: shortener-db
          image: postgres:16.0
          ports:
            - containerPort: 5432
          env:
            - name: POSTGRES_USER
              value: "user"
            - name: POSTGRES_PASSWORD
              value: "password"
            - name: POSTGRES_DB
              value: "shortener_db"
```

Next, you need a secret for your db to provide it to UrlShortener operator:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: storage-dsn
type: Opaque
stringData:
  dsn: "postgresql://user:password@shortener-db/shortener_db?sslmode=disable"
```

#### Network

Operator provides Service of type ClusterIP.  
You are free to choose your way providing public access to your instance of application. For example:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: shortener-ingress
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
spec:
  rules:
    - http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: urlshortener-sample
                port:
                  number: 8080
```

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

