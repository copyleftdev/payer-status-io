# Deployment Guide

## Table of Contents
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Systemd Service](#systemd-service)
- [Configuration Management](#configuration-management)
- [High Availability](#high-availability)
- [Backup & Recovery](#backup--recovery)
- [Monitoring](#monitoring)
- [Scaling](#scaling)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU       | 1 core  | 2+ cores    |
| Memory    | 512MB   | 2GB+        |
| Storage   | 100MB   | 1GB+        |
| Network   | 100Mbps | 1Gbps+      |


### Software Requirements

- Linux/Unix or Windows Server 2019+
- Docker 20.10+ (for containerized deployment)
- Kubernetes 1.20+ (for Kubernetes deployment)
- 1GB free disk space
- 1GB free memory
- Open ports: 8080 (HTTP/WebSocket), 9090 (metrics)

## Quick Start

### Binary Deployment

1. Download the latest release for your platform from the [releases page](https://github.com/copyleftdev/payer-status-io/releases)

2. Make the binary executable:
   ```bash
   chmod +x payer-status-io-linux-amd64
   ```

3. Create a configuration file (`config.yaml`) - see [CONFIGURATION.md](CONFIGURATION.md)

4. Run the application:
   ```bash
   ./payer-status-io-linux-amd64 --config config.yaml
   ```

### Verify Installation

```bash
# Check HTTP server
curl http://localhost:8080/health

# Check metrics
curl http://localhost:9090/metrics
```

## Docker Deployment

### Using Docker Compose (Recommended)

1. Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  payer-status-io:
    image: ghcr.io/copyleftdev/payer-status-io:latest
    container_name: payer-status-io
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./config:/app/config:ro
    environment:
      - CONFIG_PATH=/app/config/config.yaml
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

2. Create a `config` directory and add your `config.yaml` file

3. Start the service:
   ```bash
   docker-compose up -d
   ```

### Using Docker Directly

```bash
docker run -d \
  --name payer-status-io \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/config.yaml \
  ghcr.io/copyleftdev/payer-status-io:latest
```

### Building from Source

```bash
# Build the Docker image
docker build -t payer-status-io .

# Run the container
docker run -d \
  --name payer-status-io \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/config.yaml \
  payer-status-io
```

## Kubernetes Deployment

### Prerequisites

- kubectl configured to communicate with your cluster
- Helm 3.0+ (optional)

### Using Helm (Recommended)

1. Add the Helm repository:
   ```bash
   helm repo add payer-status-io https://copyleftdev.github.io/payer-status-io-helm
   helm repo update
   ```

2. Create a values.yaml file:
   ```yaml
   replicaCount: 3
   
   image:
     repository: ghcr.io/copyleftdev/payer-status-io
     tag: latest
     pullPolicy: IfNotPresent
   
   service:
     type: ClusterIP
     port: 8080
     metricsPort: 9090
   
   config: |
     # Your YAML configuration here
     server:
       port: 8080
     # ...
   
   resources:
     limits:
       cpu: 500m
       memory: 512Mi
     requests:
       cpu: 100m
       memory: 128Mi
   
   autoscaling:
     enabled: true
     minReplicas: 2
     maxReplicas: 10
     targetCPUUtilizationPercentage: 70
     targetMemoryUtilizationPercentage: 80
   ```

3. Install the chart:
   ```bash
   helm install payer-status-io payer-status-io/payer-status-io -f values.yaml
   ```

### Using kubectl

1. Create a ConfigMap for the configuration:
   ```yaml
   # config.yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: payer-status-io-config
   data:
     config.yaml: |
       server:
         port: 8080
       # ... rest of your config
   ```

2. Create a Deployment:
   ```yaml
   # deployment.yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: payer-status-io
     labels:
       app: payer-status-io
   spec:
     replicas: 3
     selector:
       matchLabels:
         app: payer-status-io
     template:
       metadata:
         labels:
           app: payer-status-io
       spec:
         containers:
         - name: payer-status-io
           image: ghcr.io/copyleftdev/payer-status-io:latest
           ports:
           - containerPort: 8080
             name: http
           - containerPort: 9090
             name: metrics
           volumeMounts:
           - name: config
             mountPath: /app/config
           resources:
             limits:
               cpu: 500m
               memory: 512Mi
             requests:
               cpu: 100m
               memory: 128Mi
           livenessProbe:
             httpGet:
               path: /health
               port: http
             initialDelaySeconds: 10
             periodSeconds: 10
           readinessProbe:
             httpGet:
               path: /health
               port: http
             initialDelaySeconds: 5
             periodSeconds: 5
         volumes:
         - name: config
           configMap:
             name: payer-status-io-config
             items:
             - key: config.yaml
               path: config.yaml
   ```

3. Create a Service:
   ```yaml
   # service.yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: payer-status-io
   spec:
     selector:
       app: payer-status-io
     ports:
     - name: http
       port: 8080
       targetPort: http
     - name: metrics
       port: 9090
       targetPort: metrics
     type: ClusterIP
   ```

4. Apply the manifests:
   ```bash
   kubectl apply -f config.yaml
   kubectl apply -f deployment.yaml
   kubectl apply -f service.yaml
   ```

### Ingress Configuration

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: payer-status-io
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$1
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - status.example.com
    secretName: payer-status-io-tls
  rules:
  - host: status.example.com
    http:
      paths:
      - path: /(.*)
        pathType: Prefix
        backend:
          service:
            name: payer-status-io
            port:
              number: 8080
```

## Systemd Service

### Create a systemd Service File

```ini
# /etc/systemd/system/payer-status-io.service
[Unit]
Description=Payer Status IO
After=network.target

[Service]
Type=simple
User=payerstatus
Group=payerstatus
WorkingDirectory=/opt/payer-status-io
ExecStart=/usr/local/bin/payer-status-io --config /etc/payer-status-io/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

[Install]
WantedBy=multi-user.target
```

### Setup Instructions

1. Create a dedicated user:
   ```bash
   sudo useradd --system --shell /bin/false --home-dir /nonexistent --no-create-home payerstatus
   ```

2. Create directories:
   ```bash
   sudo mkdir -p /opt/payer-status-io
   sudo mkdir -p /etc/payer-status-io
   sudo mkdir -p /var/log/payer-status-io
   ```

3. Install the binary:
   ```bash
   sudo cp payer-status-io-linux-amd64 /usr/local/bin/payer-status-io
   sudo chmod +x /usr/local/bin/payer-status-io
   ```

4. Copy the configuration:
   ```bash
   sudo cp config.yaml /etc/payer-status-io/
   sudo chown -R payerstatus:payerstatus /etc/payer-status-io
   ```

5. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable payer-status-io
   sudo systemctl start payer-status-io
   ```

6. Check the status:
   ```bash
   sudo systemctl status payer-status-io
   sudo journalctl -u payer-status-io -f
   ```

## Configuration Management

### Environment-Based Configuration

Use environment variables to manage different environments (development, staging, production):

```bash
# .env.development
CONFIG_PATH=./config/development.yaml
LOG_LEVEL=debug

# .env.production
CONFIG_PATH=./config/production.yaml
LOG_LEVEL=info
```

### Secrets Management

For production, use a secrets manager or Kubernetes secrets:

```bash
# Create a Kubernetes secret
kubectl create secret generic payer-status-secrets \
  --from-literal=database-password='your-password' \
  --from-literal=api-key='your-api-key'
```

Reference in your configuration:

```yaml
database:
  password: ${DATABASE_PASSWORD}
  
api:
  key: ${API_KEY}
```

## High Availability

### Multi-Instance Deployment

Deploy multiple instances behind a load balancer:

1. Use a shared database or distributed cache for state
2. Configure session affinity if needed
3. Use a service mesh for service discovery

### Database Replication

If using a database:

1. Set up primary-replica replication
2. Configure read replicas for read-heavy workloads
3. Implement connection pooling

### Circuit Breakers

Use a service mesh like Istio or Linkerd to implement circuit breakers:

```yaml
# Istio VirtualService example
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: payer-status-io
spec:
  hosts:
  - status.example.com
  http:
  - route:
    - destination:
        host: payer-status-io
        subset: v1
    retries:
      attempts: 3
      perTryTimeout: 2s
    timeout: 10s
```

## Backup & Recovery

### Configuration Backup

```bash
# Backup configuration
sudo tar -czvf payer-status-io-backup-$(date +%Y%m%d).tar.gz /etc/payer-status-io /var/lib/payer-status-io

# Schedule daily backups
sudo crontab -e
# Add:
0 2 * * * tar -czf /backups/payer-status-io-$(date +\%Y\%m\%d).tar.gz /etc/payer-status-io /var/lib/payer-status-io
```

### Database Backup

If using a database, implement regular backups:

```bash
# Example PostgreSQL backup
pg_dump -U postgres payer_status_db > payer_status_backup_$(date +%Y%m%d).sql

# Schedule with cron
0 1 * * * pg_dump -U postgres payer_status_db > /backups/payer_status_db_$(date +\%Y\%m\%d).sql
```

## Monitoring

### Prometheus & Grafana

1. Deploy Prometheus and Grafana
2. Configure Prometheus to scrape metrics from `:9090/metrics`
3. Import the provided Grafana dashboard

### Logging

Use a centralized logging solution:

- ELK Stack (Elasticsearch, Logstash, Kibana)
- Loki + Grafana
- AWS CloudWatch
- Google Cloud Logging

### Alerting

Set up alerts for:

- High error rates
- Increased latency
- Service unavailability
- Resource constraints

## Scaling

### Horizontal Scaling

1. Stateless Design: Ensure the application is stateless
2. Session Management: Use Redis or similar for distributed sessions
3. Caching: Implement caching for frequently accessed data

### Vertical Scaling

1. Increase CPU/Memory based on metrics
2. Optimize database queries
3. Implement connection pooling

### Auto-Scaling

For Kubernetes:

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: payer-status-io
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: payer-status-io
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Security

### Network Security

1. Use TLS for all communications
2. Implement network policies
3. Use a Web Application Firewall (WAF)

### Access Control

1. Implement authentication and authorization
2. Use RBAC (Role-Based Access Control)
3. Implement rate limiting

### Security Headers

Add security headers to responses:

```yaml
# In your configuration
server:
  security_headers:
    enable: true
    sts_seconds: 31536000
    sts_include_subdomains: true
    sts_preload: true
    x_frame_options: DENY
    x_content_type_options: nosniff
    x_xss_protection: "1; mode=block"
    referrer_policy: strict-origin-when-cross-origin
    content_security_policy: "default-src 'self';"
    permissions_policy: "geolocation=(), microphone=(), camera=()"
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   sudo lsof -i :8080
   sudo kill -9 <PID>
   ```

2. **Permission Denied**
   ```bash
   sudo chown -R payerstatus:payerstatus /var/log/payer-status-io
   ```

3. **Service Not Starting**
   ```bash
   sudo journalctl -u payer-status-io -f
   ```

4. **High Memory Usage**
   - Check for memory leaks
   - Adjust JVM settings if applicable
   - Increase memory limits

### Debugging

1. Enable debug logging:
   ```yaml
   logging:
     level: debug
   ```

2. Use pprof for profiling:
   ```bash
   # Start with pprof
   ./payer-status-io --pprof :6060
   
   # Generate CPU profile
   go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
   
   # Generate memory profile
   go tool pprof http://localhost:6060/debug/pprof/heap
   ```

### Getting Help

1. Check the logs:
   ```bash
   sudo journalctl -u payer-status-io -f
   ```

2. Check the documentation
3. Open an issue on GitHub
4. Contact support

## Maintenance

### Upgrading

1. Backup your configuration and data
2. Check the release notes for breaking changes
3. Follow the upgrade guide for your deployment method
4. Test in a staging environment first

### Monitoring

Regularly monitor:

- CPU and memory usage
- Disk I/O
- Network traffic
- Error rates
- Response times

### Patching

- Apply security patches promptly
- Schedule regular maintenance windows
- Test patches in staging before production
