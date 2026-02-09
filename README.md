<div align="center">

<img src="https://pub-9030d6e053cc40b380e0f63662daf8ed.r2.dev/logo.png" alt="THISJOWI Logo" width="150"/>

# THISJOWI Server

**Microservices backend with Spring Boot + Kubernetes**

[![Java 21](https://img.shields.io/badge/Java-21-ED8B00?style=flat-square&logo=openjdk)](https://openjdk.org/) [![Spring Boot](https://img.shields.io/badge/Spring%20Boot-3.2.5-6DB33F?style=flat-square&logo=spring)](https://spring.io/) [![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-326CE5?style=flat-square&logo=kubernetes)](https://kubernetes.io/) [![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker)](https://www.docker.com/)

</div>

## 🚀 Quick Start

### Prerequisites
- Java 21+
- Docker & Docker Compose
- PostgreSQL, Redis, Kafka (or use Docker)

### Run with Docker Compose (Recommended)

```bash
git clone https://github.com/THISJOWI/THISJOWI-Server.git
cd THISJOWI-Server

# Start infrastructure
docker-compose up -d postgres redis kafka

# Start services (in order)
cd Eureka && ./mvnw spring-boot:run &        # Service discovery
cd ../Authentication && ./mvnw spring-boot:run &  # Auth service
cd ../Notes && ./mvnw spring-boot:run &      # Notes service
cd ../Messages && ./mvnw spring-boot:run &   # Messages service
cd ../Cloud && ./mvnw spring-boot:run        # API Gateway
```

### Access Points

| Service | URL | Description |
|---------|-----|-------------|
| **API Gateway** | http://localhost:8100 | Main entry point |
| **Eureka Dashboard** | http://localhost:8761 | Service registry |
| **Auth API** | http://localhost:8082/swagger-ui.html | Auth docs |
| **Notes API** | http://localhost:8083/swagger-ui.html | Notes docs |

### Test the API

```bash
# Register a user
curl -X POST http://localhost:8100/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test@example.com","password":"Test123!"}'

# Login and get JWT token
curl -X POST http://localhost:8100/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test@example.com","password":"Test123!"}'

# Create a note (use JWT from login)
curl -X POST http://localhost:8100/api/v1/notes \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Note","content":"Hello World"}'
```

---

## 📦 What's Included

**Services:**
- 🔐 **Authentication** - JWT auth + OAuth2 (Google, GitHub) + LDAP
- 📝 **Notes** - Encrypted notes with tags and search
- 💬 **Messages** - Real-time messaging with Cassandra
- 🔑 **Password** - Secure password vault
- 🔔 **OTP** - One-time password generation
- 🌐 **API Gateway** - Routes all requests
- 🔍 **Service Discovery** - Eureka registry

**Tech Stack:**
- **Backend:** Spring Boot 3.2.5, Java 21
- **Databases:** PostgreSQL, Cassandra, CockroachDB
- **Cache:** Redis/KeyDB
- **Messaging:** Apache Kafka
- **Security:** JWT, OAuth2, AES-256 encryption
- **DevOps:** Docker, Kubernetes, Helm


---

## 🏗️ Architecture

```
Client → API Gateway (8100) → Eureka (8761) → Microservices
                                          │
                                          ├─ Auth (8082) → PostgreSQL + Redis
                                          ├─ Notes (8083) → PostgreSQL + Kafka
                                          ├─ Messages (8085) → Cassandra + Kafka
                                          ├─ Password (8084) → PostgreSQL
                                          └─ OTP (8086) → Redis
```

**Flow:** Client requests hit the Gateway → Gateway discovers services via Eureka → Routes to appropriate microservice → Service handles logic & persists to database

---

## ⚙️ Configuration


### Environment Variables

Create a `.env` file or configure in `application.yaml`:

```yaml
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=your_user
DB_PASSWORD=your_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Kafka
KAFKA_HOST=localhost:9092

# JWT
JWT_SECRET=your-secret-key-min-32-characters
```

Generate JWT secret:
```bash
openssl rand -base64 32
```


---

## 🚀 Kubernetes Deployment

```bash
# 1. Configure secrets
cp infrastructure/k8s/utils/secret.example.yaml infrastructure/k8s/utils/secret.yaml
nano infrastructure/k8s/utils/secret.yaml  # Edit with your values

# 2. Deploy infrastructure
kubectl apply -f infrastructure/k8s/databases/
kubectl apply -f infrastructure/k8s/utils/

# 3. Deploy services
kubectl apply -f infrastructure/k8s/application/

# 4. Check status
kubectl get pods
kubectl get services

# 5. Access services
kubectl port-forward svc/gateway-service 8100:8100
kubectl port-forward svc/eureka-service 8761:8761
```


## 🐞 Troubleshooting

**Service not registering with Eureka:**
- Wait 30-60s for registration
- Check Eureka is running: `curl http://localhost:8761`
- Verify `eureka.client.serviceUrl` in `application.yaml`

**Database connection failed:**
- Check PostgreSQL is running: `docker ps | grep postgres`
- Verify credentials in `.env` or `secret.yaml`
- Test connection: `psql -h localhost -U your_user`

**JWT validation failed (401 errors):**
- Verify JWT secret matches across all services
- Check token format: `Authorization: Bearer <token>`
- Verify Redis cache is accessible

**Port already in use:**
```bash
# Find process
lsof -i :8082  # Mac/Linux
netstat -ano | findstr :8082  # Windows
```


---

## 📚 Documentation & Support

- **Swagger API Docs:** http://localhost:8082/swagger-ui.html (Auth), http://localhost:8083/swagger-ui.html (Notes)
- **GitHub Issues:** Report bugs and request features
- **Website:** [thisjowi.uk](https://thisjowi.uk) - Complete documentation and tutorials

---

## 📝 License

Proprietary License - See [LICENCE.md](LICENCE.md). Commercial use within your organization allowed. Redistribution requires authorization.

---

<div align="center">

**Made with ❤️ by THISJOWI**

[🌐 Website](https://thisjowi.uk) • [🐛 Issues](../../issues) • [🤝 Contributing](CONTRIBUTING.md)

</div>

```bash
cd Eureka
./mvnw spring-boot:run

# Wait for startup (check logs for "Eureka Server is ready")
# Access dashboard: http://localhost:8761
```

#### 2️⃣ Start Authentication Service

```bash
cd Authentication
./mvnw spring-boot:run -Dspring-boot.run.profiles=dev

# Verify registration in Eureka dashboard
# Access Swagger: http://localhost:8082/swagger-ui.html
```

#### 3️⃣ Start Notes Service

```bash
cd Notes
./mvnw spring-boot:run

# Verify: http://localhost:8083/actuator/health
```

#### 4️⃣ Start Password Service

```bash
cd Password
./mvnw spring-boot:run

# Verify: http://localhost:8084/actuator/health
```

#### 5️⃣ Start Cloud Gateway

```bash
cd Cloud
./mvnw spring-boot:run

# Gateway ready at: http://localhost:8100
```

### Access Points

| Service | URL | Description |
|---------|-----|-------------|
| 🌐 **API Gateway** | http://localhost:8100 | Main entry point for all requests |
| 🔍 **Eureka Dashboard** | http://localhost:8761 | Service registry and health status |
| 🔐 **Auth API** | http://localhost:8082/api/auth | Direct authentication endpoints |
| 🔐 **Auth Swagger** | http://localhost:8082/swagger-ui.html | Auth API documentation |
| 📝 **Notes API** | http://localhost:8083/api/v1/notes | Direct notes endpoints |
| 🔑 **Password API** | http://localhost:8084/api/v1/passwords | Direct password endpoints |

### Using Docker Compose

```bash
# Start all services with one command
docker-compose up -d

# View logs
docker-compose logs -f [service-name]

# Stop all services
docker-compose down

# Rebuild and start
docker-compose up -d --build
```

### Using Kubernetes

```bash
# Check all pods are running
kubectl get pods

# View specific service logs
kubectl logs -f deployment/auth-service

# Port forward to access locally
kubectl port-forward svc/eureka-service 8761:8761
kubectl port-forward svc/gateway-service 8100:8100

# Scale a service
kubectl scale deployment/notes-service --replicas=3

# Check service status
kubectl get services
kubectl describe pod <pod-name>
```

---

## 🚢 Deployment

### Docker Deployment

```bash
# Build Docker images
cd Authentication
docker build -t thisjowi/auth:latest .

cd ../Notes
docker build -t thisjowi/notes:latest .

# Push to registry
docker push thisjowi/auth:latest
docker push thisjowi/notes:latest
```

### Kubernetes Deployment

1. **Configure Secrets**
   ```bash
   kubectl apply -f kubernetes/utils/secret.yaml
   ```

2. **Deploy Databases**
   ```bash
   kubectl apply -f kubernetes/databases/cockroachdb.yaml
   kubectl apply -f kubernetes/databases/keydb.yaml
   ```

3. **Deploy Infrastructure**
   ```bash
   kubectl apply -f kubernetes/utils/kafka.yaml
   ```

4. **Deploy Services**
   ```bash
   kubectl apply -f kubernetes/application/
   ```

5. **Verify Deployment**
   ```bash
   kubectl get all
   kubectl logs -f deployment/auth-service
   ```

## 📚 API Documentation

### Swagger UI

Access API documentation at:
- **Authentication:** `http://localhost:8082/swagger-ui.html`
- **Notes:** `http://localhost:8083/swagger-ui.html`
- **Password:** `http://localhost:8084/swagger-ui.html`

### Example Requests

#### Authentication

```bash
# Register a new user
curl -X POST http://localhost:8100/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@example.com",
    "password": "SecurePass123!"
  }'

# Login
curl -X POST http://localhost:8100/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@example.com",
    "password": "SecurePass123!"
  }'
```

#### Notes

```bash
# Create a note (requires JWT token)
curl -X POST http://localhost:8100/api/v1/notes \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My Note",
    "content": "This is encrypted content"
  }'
```

## 💻 Development

### Project Structure

```
backend/
├── Authentication/       # Authentication & User Management
├── Notes/               # Notes Management Service
├── Password/            # Password Vault Service
├── Cloud/               # API Gateway
├── Eureka/              # Service Discovery
└── kubernetes/          # K8s deployment configs
    ├── application/     # Service deployments
    ├── databases/       # Database configs
    ├── utils/          # Utilities (Kafka, secrets)
    └── templates/      # Reusable templates
```

### Code Style

This project follows:
- Java Code Conventions
- Spring Boot Best Practices
- Clean Code principles

### Git Workflow

```bash
# Create a feature branch
git checkout -b feature/your-feature-name

# Make changes and commit
git add .
git commit -m "feat: add new feature"

# Push and create PR
git push origin feature/your-feature-name
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## 🧪 Testing

### Run Unit Tests

```bash
# All services
./test-all.sh

# Individual service
cd Authentication
./mvnw test
```

### Integration Tests

```bash
cd Authentication
./mvnw verify
```

### Code Coverage

```bash
./mvnw clean test jacoco:report
# Report available at: target/site/jacoco/index.html
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on:

- Code of Conduct
- Development process
- Pull request guidelines
- Issue reporting

## 🔒 Security

Security is a top priority. Please:

- Never commit secrets or credentials
- Review [SECURITY.md](SECURITY.md) before deployment
- Report security issues privately to: [security@thisjowi.uk]

### Security Features

- ✅ JWT-based authentication
- ✅ AES-256 encryption for sensitive data
- ✅ Secrets managed via Kubernetes Secrets
- ✅ HTTPS ready
- ✅ CORS configuration
- ✅ Input validation

## 📄 License

This project is licensed under the **THISJOWI Proprietary Source License**.

- ✅ **Allowed:** Personal use, commercial use within your organization, modifications for internal use
- ❌ **Restricted:** Redistribution, SaaS offerings, sublicensing without authorization
- 💰 **Redistribution:** Requires separate agreement and royalties

See the [LICENCE.md](LICENCE.md) file for complete terms and conditions.

For redistribution inquiries: **contact@thisjowi.uk**

---

## � Troubleshooting

<details>
<summary><b>Service Not Registering with Eureka</b></summary>

**Problem:** Service doesn't appear in Eureka dashboard

**Solutions:**
1. Verify Eureka is running on port 8761
2. Check `eureka.client.serviceUrl.defaultZone` in `application.yaml`
3. Wait 30-60 seconds for registration to complete
4. Check service logs for connection errors:
   ```bash
   docker logs [container-name]
   ```

</details>

<details>
<summary><b>Database Connection Failed</b></summary>

**Problem:** Cannot connect to PostgreSQL

**Solutions:**
1. Verify PostgreSQL is running: `docker ps | grep postgres`
2. Check connection string in `application.yaml`
3. Verify database exists: `psql -U postgres -c "\l"`
4. Check firewall rules allow port 5432
5. Verify credentials in secrets

</details>

<details>
<summary><b>JWT Token Validation Failed</b></summary>

**Problem:** 401 Unauthorized errors

**Solutions:**
1. Verify JWT secret is the same across all services
2. Check token hasn't expired
3. Ensure `Authorization: Bearer <token>` header format
4. Verify Redis cache is accessible
5. Check logs for specific validation errors

</details>

<details>
<summary><b>Kafka Connection Issues</b></summary>

**Problem:** Services can't connect to Kafka

**Solutions:**
1. Verify Kafka is running: `docker ps | grep kafka`
2. Check `bootstrap-servers` configuration
3. Test connectivity: `kafka-topics --bootstrap-server localhost:9092 --list`
4. Create required topics if missing

</details>

<details>
<summary><b>Port Already in Use</b></summary>

**Problem:** `Address already in use` error

**Solutions:**
1. Find process using the port: `lsof -i :8082` (Mac/Linux) or `netstat -ano | findstr :8082` (Windows)
2. Kill the process or change service port in `application.yaml`

</details>

---

## �💬 Support

Need help? Here are your options:

### 📖 Documentation
- **README** - You're reading it!
- **[SECURITY.md](SECURITY.md)** - Security configuration guide
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - How to contribute
- **Swagger UI** - API documentation at `/swagger-ui.html` endpoints

### 🐛 Issues & Bugs
1. Search [existing issues](../../issues) to avoid duplicates
2. Create a [new issue](../../issues/new) with:
   - Clear description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Java version, etc.)
   - Relevant logs

### 💡 Feature Requests
- Open a [GitHub Discussion](../../discussions)
- Describe the use case and expected behavior
- Explain why it would be useful

### 🌐 Website & Documentation

For comprehensive guides, tutorials, and detailed documentation, visit:

**👉 [thisjowi.uk](https://thisjowi.uk)**

You'll find:
- 📖 Complete API documentation
- 🎥 Video tutorials
- 🏗 Architecture deep-dives
- 🔐 Security best practices
- 💡 Implementation examples
- 🚀 Advanced deployment guides
- 📰 Latest updates and blog posts
- 🎓 Training materials

### 💼 Professional Support

For commercial support, consulting, or custom development:

📧 **Email:** dev@thisjowi.uk  
🌐 **Website:** [thisjowi.uk](https://thisjowi.uk)  
💼 **Business Inquiries:** contact@thisjowi.uk

---

## 🙏 Acknowledgments

- [Spring Boot Team](https://spring.io/projects/spring-boot) for the excellent framework
- [Netflix OSS](https://netflix.github.io/) for Eureka and other cloud tools
- [HashiCorp](https://www.hashicorp.com/) for Vault
- [Apache Kafka](https://kafka.apache.org/) team
- The open-source community for inspiration and tools

---

## 📞 Contact

- 📧 **Email:** support@thisjowi.uk
- 🐛 **Issues:** [GitHub Issues](../../issues)
- 💬 **Discussions:** [GitHub Discussions](../../discussions)
- 🌐 **Website:** [thisjowi.uk](https://thisjowi.uk)

---

<div align="center">

### ⭐ Star this repository if you find it helpful!

**Made with ❤️ by THISJowi**

[🌐 Website](https://thisjowi.uk) • [📚 Documentation](https://thisjowi.uk) • [🐛 Issues](../../issues) • [🤝 Contributing](CONTRIBUTING.md) • [� License](LICENCE.md)

[Back to Top](#-thisjowi-backend)

</div>
