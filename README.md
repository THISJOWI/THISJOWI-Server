<div align="center">

<img src="https://pub-9030d6e053cc40b380e0f63662daf8ed.r2.dev/logo.png" alt="THISJOWI Logo" width="150"/>

# THISJOWI Server

**Microservices backend con Spring Boot + Kubernetes**

[![Java 21](https://img.shields.io/badge/Java-21-ED8B00?style=flat-square&logo=openjdk)](https://openjdk.org/) [![Spring Boot](https://img.shields.io/badge/Spring%20Boot-3.2.5-6DB33F?style=flat-square&logo=spring)](https://spring.io/) [![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-326CE5?style=flat-square&logo=kubernetes)](https://kubernetes.io/) [![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker)](https://www.docker.com/)

</div>

## 🚀 Inicio Rápido

```bash
# Clonar repositorio
git clone https://github.com/THISJOWI/THISJOWI-Server.git
cd THISJOWI-Server

# Iniciar infraestructura
docker-compose up -d postgres redis kafka

# Iniciar servicios
cd Eureka && ./mvnw spring-boot:run &
cd ../Authentication && ./mvnw spring-boot:run &
cd ../Notes && ./mvnw spring-boot:run &
cd ../Cloud && ./mvnw spring-boot:run
```

**Acceso:**
- 🌐 API Gateway: http://localhost:8100
- 🔍 Eureka: http://localhost:8761
- 📖 Swagger Auth: http://localhost:8082/swagger-ui.html
- 📖 Swagger Notes: http://localhost:8083/swagger-ui.html

---

## 📦 Servicios

| Servicio | Puerto | Descripción |
|----------|--------|-------------|
| 🔐 **Authentication** | 8082 | JWT + OAuth2 + LDAP |
| 📝 **Notes** | 8083 | Notas encriptadas con tags |
| 💬 **Messages** | 8085 | Mensajería real-time |
| 🔑 **Password** | 8084 | Bóveda de contraseñas |
| 🔔 **OTP** | 8086 | Generación de OTP |
| 🌐 **Gateway** | 8100 | Enrutamiento de peticiones |
| 🔍 **Eureka** | 8761 | Registro de servicios |

**Stack:**  
Java 21 • Spring Boot 3.2.5 • PostgreSQL • Cassandra • Redis • Kafka • Docker • Kubernetes

---

## 🏗️ Arquitectura

```
Cliente → Gateway (8100) → Eureka (8761) → Microservicios
                              │
                              ├─ Auth → PostgreSQL + Redis
                              ├─ Notes → PostgreSQL + Kafka
                              ├─ Messages → Cassandra + Kafka
                              ├─ Password → PostgreSQL
                              └─ OTP → Redis
```

---

## 🧪 Test de API

```bash
# Registrar usuario
curl -X POST http://localhost:8100/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test@example.com","password":"Test123!"}'

# Login (obtener JWT)
curl -X POST http://localhost:8100/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test@example.com","password":"Test123!"}'

# Crear nota (usar JWT del login)
curl -X POST http://localhost:8100/api/v1/notes \
  -H "Authorization: Bearer TU_TOKEN_JWT" \
  -H "Content-Type: application/json" \
  -d '{"title":"Mi Nota","content":"Hola Mundo"}'
```

---

## ⚙️ Configuración

Crea `.env` o configura en `application.yaml`:

```yaml
# Base de datos
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=tu_usuario
DB_PASSWORD=tu_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Kafka
KAFKA_HOST=localhost:9092

# JWT (generar con: openssl rand -base64 32)
JWT_SECRET=tu-clave-secreta-min-32-caracteres
```

---

## 🐳 Docker Compose

```bash
# Iniciar todo
docker-compose up -d

# Ver logs
docker-compose logs -f [servicio]

# Detener todo
docker-compose down
```

---

## ☸️ Kubernetes

```bash
# 1. Configurar secretos
cp infrastructure/k8s/utils/secret.example.yaml infrastructure/k8s/utils/secret.yaml
nano infrastructure/k8s/utils/secret.yaml

# 2. Desplegar
kubectl apply -f infrastructure/k8s/databases/
kubectl apply -f infrastructure/k8s/utils/
kubectl apply -f infrastructure/k8s/application/

# 3. Verificar
kubectl get pods
kubectl get services

# 4. Acceder
kubectl port-forward svc/gateway-service 8100:8100
kubectl port-forward svc/eureka-service 8761:8761
```

---

## 🔧 Troubleshooting

**Servicio no se registra en Eureka**
- Esperar 30-60s para el registro
- Verificar: `curl http://localhost:8761`
- Revisar `eureka.client.serviceUrl` en `application.yaml`

**Error de conexión a base de datos**
- Verificar PostgreSQL: `docker ps | grep postgres`
- Probar conexión: `psql -h localhost -U tu_usuario`
- Revisar credenciales en `.env` o `secret.yaml`

**Error 401 (JWT inválido)**
- Verificar que el JWT secret sea igual en todos los servicios
- Formato correcto: `Authorization: Bearer <token>`
- Verificar acceso a Redis

**Puerto en uso**
```bash
# Mac/Linux
lsof -i :8082

# Windows
netstat -ano | findstr :8082
```

---

## 📚 Documentación

- **Swagger:** http://localhost:8082/swagger-ui.html
- **Website:** [thisjowi.uk](https://thisjowi.uk) - Documentación completa
- **Issues:** [GitHub Issues](../../issues)

---

## 🔒 Seguridad

- ✅ Autenticación JWT
- ✅ Encriptación AES-256
- ✅ OAuth2 (Google, GitHub)
- ✅ Secrets en Kubernetes
- ✅ HTTPS ready

**Reportar problemas de seguridad:** security@thisjowi.uk

---

## 📝 Licencia

Licencia Propietaria - Ver [LICENCE.md](LICENCE.md)

- ✅ Uso personal y comercial interno
- ❌ Redistribución sin autorización
- 💰 Redistribución requiere acuerdo

**Consultas:** contact@thisjowi.uk

---

<div align="center">

**Hecho con ❤️ por THISJOWI**

[🌐 Website](https://thisjowi.uk) • [🐛 Issues](../../issues) • [🤝 Contributing](CONTRIBUTING.md)

⭐ ¡Dale una estrella si te resulta útil!

</div>
