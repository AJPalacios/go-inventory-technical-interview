# 🚀 Production Deployment Guide

**Complete Guide for Deploying the Distributed Inventory Management System**

---

## 🎯 Overview

This guide provides step-by-step instructions for deploying the inventory management system across different environments: Development, Staging, and Production.

### 🏗️ Architecture Overview

```
Development     →    Staging        →    Production
┌────────────┐   ┌────────────┐   ┌────────────┐
│  SQLite     │   │ PostgreSQL │   │ PostgreSQL │
│  Local      │   │ Docker     │   │ Kubernetes │
│  No Cache   │   │ Redis      │   │ Redis      │
│  File Logs  │   │ Structured │   │ Prometheus │
└────────────┘   └────────────┘   └────────────┘
```

---

## 📝 Prerequisites

### 🛠️ Development Environment
```bash
# Required tools
go version              # Go 1.21+
docker version          # Docker 20.10+
docker-compose version  # Docker Compose 2.0+
kubectl version         # Kubernetes CLI (for production)

# Optional tools
make --version          # Build automation
curl --version          # API testing
jq --version            # JSON processing
```

### 📚 Environment Variables
```bash
# Create app.env file
cp app.env.example app.env

# Essential configuration
ENV=development|staging|production
SERVER_PORT=8080
DATABASE_DRIVER=sqlite3|postgres
DATABASE_DSN=./inventory.db|postgres://...
LOG_LEVEL=debug|info|warn|error
LOG_FORMAT=json|console
```

---

## 🖥️ Development Deployment

### 1️⃣ Local Setup

```bash
# Clone and setup
git clone <repository-url>
cd inventory
go mod tidy

# Create database
make createdb

# Run database migrations
make migrateup

# Start development server
make server

# Verify health
curl http://localhost:8080/health
```

### 2️⃣ Development Configuration

```bash
# app.env for development
ENV=development
SERVER_PORT=8080
DATABASE_DRIVER=sqlite3
DATABASE_DSN=./inventory.db?_journal_mode=WAL&_foreign_keys=on
LOG_LEVEL=debug
LOG_FORMAT=console
METRICS_ENABLED=false
CACHE_ENABLED=false
```

### 3️⃣ Testing

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run ACID compliance tests
go test ./internal/repository/ -run "ACID|Atomic|Isolation" -v

# Benchmark tests
go test -bench=. ./internal/repository/
```

---

## 📪 Staging Deployment (Docker Compose)

### 1️⃣ Docker Compose Setup

```yaml
# docker-compose.staging.yml
version: '3.8'

services:
  inventory-api:
    build:
      context: .
      dockerfile: Dockerfile
      target: production
    ports:
      - "8080:8080"
    environment:
      - ENV=staging
      - DATABASE_DRIVER=postgres
      - DATABASE_DSN=postgres://inventory:staging_password@postgres:5432/inventory?sslmode=disable
      - REDIS_URL=redis://redis:6379/0
      - LOG_LEVEL=info
      - LOG_FORMAT=json
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: inventory
      POSTGRES_USER: inventory
      POSTGRES_PASSWORD: staging_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/init.sql:/docker-entrypoint-initdb.d/01-init.sql:ro
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U inventory -d inventory"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --requirepass staging_redis_password
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "--no-auth-warning", "-a", "staging_redis_password", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--web.enable-lifecycle'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=staging_grafana_password
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana:/etc/grafana/provisioning

volumes:
  postgres_data:
  redis_data:
  prometheus_data:
  grafana_data:

networks:
  default:
    name: inventory-staging
```

### 2️⃣ Deploy to Staging

```bash
# Build and deploy
docker-compose -f docker-compose.staging.yml up -d --build

# Check service health
docker-compose -f docker-compose.staging.yml ps

# View logs
docker-compose -f docker-compose.staging.yml logs -f inventory-api

# Run database migrations
docker-compose -f docker-compose.staging.yml exec inventory-api \
  ./inventory migrate up

# Health check
curl http://localhost:8080/health

# Metrics endpoint
curl http://localhost:9090  # Prometheus
curl http://localhost:3000  # Grafana
```

### 3️⃣ Staging Validation

```bash
# Load test with real data
cd test-api
# Use inventory-api.http with VS Code REST Client

# Database connectivity test
docker-compose -f docker-compose.staging.yml exec postgres \
  psql -U inventory -d inventory -c "SELECT COUNT(*) FROM products;"

# Redis connectivity test
docker-compose -f docker-compose.staging.yml exec redis \
  redis-cli -a staging_redis_password ping
```

---

## ⚙️ Production Deployment (Kubernetes)

### 1️⃣ Kubernetes Manifests

#### **Namespace**
```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: inventory-prod
  labels:
    app: inventory
    environment: production
```

#### **ConfigMap**
```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: inventory-config
  namespace: inventory-prod
data:
  ENV: "production"
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
  METRICS_ENABLED: "true"
  CACHE_ENABLED: "true"
  SERVER_PORT: "8080"
  DATABASE_DRIVER: "postgres"
```

#### **Secrets**
```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: inventory-secrets
  namespace: inventory-prod
type: Opaque
data:
  database-dsn: <base64-encoded-connection-string>
  redis-url: <base64-encoded-redis-url>
  api-key: <base64-encoded-api-key>
```

#### **Deployment**
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inventory-api
  namespace: inventory-prod
  labels:
    app: inventory-api
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: inventory-api
  template:
    metadata:
      labels:
        app: inventory-api
        version: v1
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: inventory-api
        image: inventory:latest
        ports:
        - containerPort: 8080
          name: http
        envFrom:
        - configMapRef:
            name: inventory-config
        env:
        - name: DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: inventory-secrets
              key: database-dsn
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: inventory-secrets
              key: redis-url
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 30
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - inventory-api
              topologyKey: kubernetes.io/hostname
```

#### **Service**
```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: inventory-service
  namespace: inventory-prod
  labels:
    app: inventory-api
spec:
  selector:
    app: inventory-api
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
  type: ClusterIP
```

#### **Ingress**
```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: inventory-ingress
  namespace: inventory-prod
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "1000"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
spec:
  tls:
  - hosts:
    - api.inventory.company.com
    secretName: inventory-tls
  rules:
  - host: api.inventory.company.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: inventory-service
            port:
              number: 80
```

### 2️⃣ Production Deployment Steps

```bash
# 1. Create namespace
kubectl apply -f k8s/namespace.yaml

# 2. Create secrets (replace with actual values)
kubectl create secret generic inventory-secrets \
  --from-literal=database-dsn="postgres://user:pass@host:5432/inventory?sslmode=require" \
  --from-literal=redis-url="redis://redis-host:6379/0" \
  --from-literal=api-key="your-api-key" \
  --namespace=inventory-prod

# 3. Apply configuration
kubectl apply -f k8s/configmap.yaml

# 4. Deploy application
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml

# 5. Verify deployment
kubectl get pods -n inventory-prod
kubectl get services -n inventory-prod
kubectl get ingress -n inventory-prod

# 6. Check logs
kubectl logs -f deployment/inventory-api -n inventory-prod

# 7. Health check
kubectl port-forward service/inventory-service 8080:80 -n inventory-prod &
curl http://localhost:8080/health
```

### 3️⃣ Production Monitoring

```bash
# Check pod status
kubectl get pods -n inventory-prod -w

# View resource usage
kubectl top pods -n inventory-prod

# Check ingress
kubectl describe ingress inventory-ingress -n inventory-prod

# Scale deployment
kubectl scale deployment inventory-api --replicas=5 -n inventory-prod

# Rolling update
kubectl set image deployment/inventory-api inventory-api=inventory:v2 -n inventory-prod
kubectl rollout status deployment/inventory-api -n inventory-prod

# Rollback if needed
kubectl rollout undo deployment/inventory-api -n inventory-prod
```

---

## 📊 Database Migration Strategy

### 🔄 SQLite to PostgreSQL Migration

```bash
# 1. Export SQLite data
sqlite3 inventory.db <<EOF
.headers on
.mode csv
.output products.csv
SELECT * FROM products;
.output inventory_items.csv
SELECT * FROM inventory_items;
.output reservations.csv
SELECT * FROM reservations;
.quit
EOF

# 2. Create PostgreSQL database
createdb inventory_prod

# 3. Run migrations
migrate -path db/migrations -database "postgres://user:pass@host:5432/inventory_prod?sslmode=require" up

# 4. Import data
psql -d inventory_prod -c "\COPY products FROM 'products.csv' CSV HEADER;"
psql -d inventory_prod -c "\COPY inventory_items FROM 'inventory_items.csv' CSV HEADER;"
psql -d inventory_prod -c "\COPY reservations FROM 'reservations.csv' CSV HEADER;"

# 5. Update sequences
psql -d inventory_prod -c "
SELECT setval(pg_get_serial_sequence('inventory_items', 'id'), 
              COALESCE(MAX(id::integer), 1)) 
FROM inventory_items WHERE id ~ '^[0-9]+$';
"

# 6. Verify data integrity
psql -d inventory_prod -c "
SELECT 
  (SELECT COUNT(*) FROM products) as products,
  (SELECT COUNT(*) FROM inventory_items) as inventory_items,
  (SELECT COUNT(*) FROM reservations) as reservations;
"
```

---

## 📊 Monitoring & Observability

### 📊 Prometheus Configuration

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "inventory_rules.yml"

scrape_configs:
  - job_name: 'inventory-api'
    static_configs:
      - targets: ['inventory-service:80']
    metrics_path: '/metrics'
    scrape_interval: 10s
    scrape_timeout: 5s
    
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']
      
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### 🎨 Grafana Dashboards

```json
# monitoring/grafana/dashboards/inventory.json
{
  "dashboard": {
    "title": "Inventory Management System",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      },
      {
        "title": "Database Connections",
        "type": "singlestat",
        "targets": [
          {
            "expr": "database_connections_active",
            "legendFormat": "Active Connections"
          }
        ]
      }
    ]
  }
}
```

---

## 🚑 Troubleshooting Guide

### ⚠️ Common Issues

#### **Database Connection Issues**
```bash
# Check connection
kubectl exec -it deployment/inventory-api -n inventory-prod -- \
  ./inventory db ping

# Check PostgreSQL logs
kubectl logs postgresql-0 -n inventory-prod

# Test connection manually
kubectl exec -it deployment/inventory-api -n inventory-prod -- \
  psql $DATABASE_DSN -c "SELECT 1;"
```

#### **High Memory Usage**
```bash
# Check resource usage
kubectl top pods -n inventory-prod

# Increase memory limits
kubectl patch deployment inventory-api -n inventory-prod -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "inventory-api",
          "resources": {
            "limits": {
              "memory": "512Mi"
            }
          }
        }]
      }
    }
  }
}'
```

#### **Version Conflicts (High Retry Rate)**
```bash
# Check retry metrics
curl http://localhost:8080/metrics | grep retry

# Increase replica count for better load distribution
kubectl scale deployment inventory-api --replicas=5 -n inventory-prod

# Check database connection pool
kubectl exec -it deployment/inventory-api -n inventory-prod -- \
  ./inventory db stats
```

### 📊 Health Checks

```bash
# Application health
curl -f http://api.inventory.company.com/health

# Database health
curl -f http://api.inventory.company.com/health/db

# Cache health  
curl -f http://api.inventory.company.com/health/cache

# Detailed health with metrics
curl -s http://api.inventory.company.com/health | jq .
```

---

## 🔄 CI/CD Pipeline

### 🚀 GitHub Actions Workflow

```yaml
# .github/workflows/deploy.yml
name: Deploy Inventory System

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run tests
      run: |
        make test
        make coverage
    
    - name: ACID Compliance Tests
      run: |
        go test ./internal/repository/ -run "ACID|Atomic|Isolation" -v
  
  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Build Docker image
      run: |
        docker build -t ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }} .
        docker build -t ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest .
    
    - name: Push to registry
      if: github.ref == 'refs/heads/main'
      run: |
        echo ${{ secrets.GITHUB_TOKEN }} | docker login ${{ env.REGISTRY }} -u ${{ github.actor }} --password-stdin
        docker push ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }}
        docker push ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
  
  deploy-staging:
    needs: build
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    environment: staging
    steps:
    - uses: actions/checkout@v4
    
    - name: Deploy to staging
      run: |
        docker-compose -f docker-compose.staging.yml pull
        docker-compose -f docker-compose.staging.yml up -d
  
  deploy-production:
    needs: build
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    environment: production
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup kubectl
      uses: azure/setup-kubectl@v3
      with:
        version: 'v1.28.0'
    
    - name: Deploy to production
      run: |
        echo "${{ secrets.KUBECONFIG }}" | base64 -d > kubeconfig
        export KUBECONFIG=kubeconfig
        
        kubectl set image deployment/inventory-api \
          inventory-api=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }} \
          -n inventory-prod
        
        kubectl rollout status deployment/inventory-api -n inventory-prod
```

---

## 📝 Operational Checklist

### ✅ Pre-Deployment Checklist

- [ ] **Environment Configuration**
  - [ ] Database connection strings updated
  - [ ] Redis configuration verified
  - [ ] SSL certificates in place
  - [ ] API keys rotated
  - [ ] Resource limits configured

- [ ] **Security**
  - [ ] Secrets properly encrypted
  - [ ] Network policies applied
  - [ ] RBAC configured
  - [ ] Ingress security headers set

- [ ] **Monitoring**
  - [ ] Prometheus scraping configured
  - [ ] Grafana dashboards imported
  - [ ] Alert rules defined
  - [ ] Log aggregation working

- [ ] **Testing**
  - [ ] Health checks passing
  - [ ] Load test executed
  - [ ] Database migration tested
  - [ ] Rollback procedure tested

### ✅ Post-Deployment Checklist

- [ ] **Verification**
  - [ ] All pods running and ready
  - [ ] Health endpoints responding
  - [ ] API functionality verified
  - [ ] Database queries working
  - [ ] Cache connectivity confirmed

- [ ] **Monitoring**
  - [ ] Metrics collection active
  - [ ] Alerts configured and tested
  - [ ] Log ingestion working
  - [ ] Dashboard data flowing

- [ ] **Performance**
  - [ ] Response times within SLA
  - [ ] Resource utilization normal
  - [ ] Connection pools healthy
  - [ ] No memory leaks detected

---

## 🎆 Summary

This deployment guide provides a comprehensive path from development to production for the inventory management system. The architecture scales from simple SQLite-based development to enterprise-grade Kubernetes deployment with full observability.

**Key Features**:
- ✅ **Progressive Deployment**: Development → Staging → Production
- ✅ **Container-Ready**: Docker with multi-stage builds
- ✅ **Kubernetes Native**: Complete K8s manifests
- ✅ **Database Migration**: SQLite to PostgreSQL path
- ✅ **Monitoring Stack**: Prometheus + Grafana + Alerting
- ✅ **CI/CD Pipeline**: Automated testing and deployment
- ✅ **Operational Excellence**: Comprehensive troubleshooting guides

---

*This deployment guide ensures a smooth transition from development to production with enterprise-grade reliability and observability.*