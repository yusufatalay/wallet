# Wallet and Asset Management Microservice System

**This project is a microservice-based system designed to manage user wallets and their crypto assets. It provides functionalities for creating wallets, handling deposits, withdrawals, and supporting both instant and scheduled transactions between wallets.**

## Architecture Overview

The system is built on a microservice architecture, with services communicating internally via high-performance ***gRPC*** calls. A dedicated ***API Gateway*** acts as the single entry point for all external clients, exposing a familiar **JSON/REST API** and routing requests to the appropriate backend services.

* **API Gateway** **: Handles all incoming HTTP requests, translates them to gRPC, and forwards them to the correct service. It is also responsible for serving the API documentation.**
* **User Service** **: Manages user registration and authentication (login), issuing JWTs for securing endpoints.**
* **Wallet Service** **: Responsible for creating, retrieving, and validating wallet ownership.**
* **Asset Service** **: Handles all financial operations, including deposits, withdrawals, and transfers (both instant and scheduled). It contains a background worker to process scheduled transactions.**

**Data and Caching:**

* **MongoDB** **: Used as the primary NoSQL database for all services. It is configured to run as a replica set to support atomic transactions.**
* **Redis** **: Implemented for caching frequently accessed data (e.g., a user's wallet list) to reduce database load and improve response times.**

## Features

* **User Management** **: Secure user registration and JWT-based authentication.**
* **Wallet Management** **: Create and list wallets, each with a unique address and network combination.**
* **Asset Operations** **:**
* **Instant Deposit/Withdrawal** **: Immediately credit or debit assets from a wallet.**
* **Instant Transfer** **: Immediately transfer assets from one user's wallet to another.**
* **Scheduled Transactions** **: Schedule a deposit, withdrawal, or transfer to be executed at a specified future time.**
* **Background Worker** **: A robust scheduler runs within the Asset Service to process all due scheduled transactions periodically.**
* **Atomic Transactions** **: All multi-step financial operations (e.g., transfers) are performed within a MongoDB transaction to ensure data integrity.**

## Tech Stack

* **Language** **: Go (Golang)**
* **Communication** **:**
* **Internal: gRPC**
* **External: REST (via grpc-gateway)**
* **Database** **: MongoDB**
* **Cache** **: Redis**
* **Containerization** **: Docker & Docker Compose**
* **API Documentation** **: Swagger (OpenAPI)**
* **Testing** **: testify/mock**
* **Linting** **: golangci-lint**

## Setup and Run

#### Prerequisites

* **Docker**
* **Docker Compose**
* **Go (v1.24+)**
* **make**
* **protoc** compiler and Go plugins (see instructions in the ```Makefile```)

#### 1. Generate Protobuf Code

**First, generate all the necessary gRPC and gateway code from the ```.proto``` definitions.**

```sh
make proto

```

#### 2. Start All Services

**Use Docker Compose to build and run all services, including the database and cache.**

```sh
docker-compose up --build

```

The system will be available at  ```http://localhost:8080```

#### 3. Initiate MongoDB Replica Set (One-Time Setup)

**To enable transaction support, MongoDB must be initiated as a replica set. After the containers are running, wait ~15 seconds and then run this command in your terminal:**

```sh
docker exec -it mongo mongosh --eval "rs.initiate({_id: 'rs0', members: [{_id: 0, host: 'mongo:27017'}]})"

```

You only need to do this once for the lifetime of the ```mongo-data``` Docker volume.

## API Documentation (Swagger UI)

**The project includes an interactive API documentation page. Once the services are running, navigate to:**

**http://localhost:8080/docs/**

You can use the dropdown menu at the top of the page to switch between services (```assets, wallets, users```). The interface allows you to test all endpoints, including authentication via the "Authorize" button.

## Testing and Linting

**The project is configured with a linter and unit tests, which can be run via the ```Makefile```.**

#### Run Linter

**To check the code quality across all services:**

```sh
make lint

```

#### Run Unit Tests

**To run the unit tests for all services:**

```sh
make test

```

## Future Improvements

**If I had more time on this project, I would focus on the following production-readiness improvements:**

* **Secret Management** **: The current ```JWT_SECRET``` is stored in plain text in the ```docker-compose.yml``` file. In a production environment, this is a major security risk. I would integrate a proper secret management solution like Kubernetes Sealed Secrets, HashiCorp Vault, or a cloud provider's secret manager (e.g., AWS Secrets Manager) to handle all sensitive credentials.**
* **Event-Driven Architecture** **: While direct gRPC calls are efficient, some cross-service communication could be further decoupled. For example, instead of the ```assets``` service directly calling the ```wallets``` service to find a wallet, the system could use a  message broker (like RabbitMQ or Kafka). The ```wallets``` service could publish "WalletCreated" or "WalletUpdated" events, and the ```assets``` service could maintain its own local, eventually consistent cache of wallet data, reducing synchronous dependencies.**
* **Comprehensive Testing** **: The current unit tests cover critical paths. I would expand this to achieve higher test coverage, including more edge cases and error conditions. Furthermore, I would add a suite of end-to-end (E2E) integration tests that test the entire user flow from the API Gateway down to the database, ensuring all services work together as expected.**
* **Observability** **: To operate this system reliably in production, I would add a comprehensive observability stack:**
* **Structured Logging** **: Implement structured logging (e.g., using zerolog or zap) in all services for easier parsing and searching.**
* **Metrics** **: Expose application-level metrics (e.g., number of transactions, RPC latency) using a library like Prometheus**
* **Distributed Tracing** **: Implement distributed tracing (e.g., using OpenTelemetry and Jaeger/Zipkin) to trace requests as they travel through the different microservices, making it easier to debug performance bottlenecks and errors.**
