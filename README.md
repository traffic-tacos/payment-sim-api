# Payment Sim API

Traffic Tacos MSA 플랫폼의 **순수 gRPC 결제 시뮬레이션 서비스**입니다. 30k RPS 처리 목표의 고성능 결제 처리를 시뮬레이션하며, AWS EventBridge + SQS를 통한 실제 비동기 메시징을 지원합니다.

## ✨ 주요 특징

🚀 **순수 gRPC 서비스** - inventory-api 패턴 준수
⚡ **실시간 비동기 처리** - EventBridge + SQS + HTTP Webhook
🎯 **실제 PG사 시뮬레이션** - 2초 지연 후 결과 처리
🔧 **grpcui 지원** - 웹 인터페이스로 쉬운 테스트
📊 **Prometheus 메트릭스** - 운영 모니터링 지원
🏥 **Kubernetes Ready** - 헬스체크 및 배포 최적화

## 🏗️ 아키텍처

**Layer 2: Business Services**에서 동작하는 결제 시뮬레이션 서비스:

- **Framework**: Go + gRPC (순수 gRPC 서비스)
- **포트**: 8030 (gRPC), 8031 (Health + Metrics)
- **핵심 기능**: 결제 Intent 생성/관리, 비동기 이벤트 처리, Webhook 콜백
- **AWS 연동**: EventBridge + SQS를 통한 실제 비동기 메시징

### Traffic Tacos MSA 포트 체계
- **Gateway API**: 8000 (HTTP 엔트리포인트)
- **Reservation API**: 8010 (Kotlin + Spring Boot)
- **Inventory API**: 8020 (Go + gRPC)
- **Payment Sim API**: 8030 (Go + gRPC) ← **현재 서비스**
- **Reservation Worker**: 8040 (백그라운드 처리)

### 이벤트 기반 아키텍처
```
Payment-Sim-API → EventBridge → SQS → Reservation-Worker
       ↓
   HTTP Webhook (병행 처리)
```

### 포트 구성 (inventory-api 패턴)
- **8030**: gRPC 서비스 (비즈니스 로직)
- **8031**: HTTP 서버 (헬스체크 + 메트릭스만)

## 🚀 빠른 시작

### 1. 의존성 설치
```bash
# Go 1.24+ 필요
go version

# gRPC 도구 설치 (필수)
go install github.com/fullstorydev/grpcui/cmd/grpcui@latest
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 개발 도구 설치 (선택적)
make dev-deps
```

### 2. 환경 설정
```bash
# 환경 변수 템플릿 복사
cp .env.template .env.local

# AWS profile 'tacos' 설정 확인
aws configure list --profile tacos

# EventBridge 및 SQS 설정 확인
aws events list-event-buses --profile tacos --region ap-northeast-2
aws sqs list-queues --profile tacos --region ap-northeast-2
```

### 3. 애플리케이션 실행
```bash
# Payment-Sim-API 실행
make build
./bin/payment-sim-api

# Reservation Worker 실행 (별도 터미널)
./bin/reservation-worker

# 또는 원스톱 실행 (스크립트 사용)
./scripts/run_local.sh start
```

### 4. 서비스 확인
```bash
# gRPC 웹 인터페이스 (추천)
grpcui -plaintext localhost:8030
# → 브라우저에서 자동으로 열림 (http://127.0.0.1:xxxxx/)

# 명령줄 테스트
grpcurl -plaintext localhost:8030 list

# 헬스체크 및 메트릭스
curl http://localhost:8031/health
curl http://localhost:8031/metrics
```

## 📚 API 문서

### gRPC 서비스

```protobuf
service PaymentService {
  rpc CreatePaymentIntent(CreatePaymentIntentRequest) returns (CreatePaymentIntentResponse);
  rpc GetPaymentIntent(GetPaymentIntentRequest) returns (GetPaymentIntentResponse);
  rpc ProcessPayment(ProcessPaymentRequest) returns (ProcessPaymentResponse);
}
```

### Health Check API

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/health` | 서비스 헬스체크 |
| GET | `/metrics` | Prometheus 메트릭스 |

### 결제 시나리오

- `approve`: 즉시 승인 (기본값)
- `fail`: 즉시 실패
- `delay`: 지연 후 승인 (기본 2초)
- `random`: 랜덤 승인/실패

## 🔧 개발 명령어

```bash
# 전체 CI 파이프라인
make ci

# 개별 명령어
make build          # 빌드
make test           # 테스트
make lint           # 린팅
make docker-build   # Docker 이미지 빌드

# 성능 테스트
make perf-test      # gRPC & REST 성능 테스트

# 개발 도구
make grpcui         # gRPC UI 시작
make status         # 서비스 상태 확인
```

## 🐳 Docker 실행

```bash
# Docker 빌드 및 실행
make docker-run

# 또는 직접 실행
docker build -t payment-sim-api:latest .
docker run -p 8030:8030 -p 8031:8031 payment-sim-api:latest
```

## ⚙️ 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `GRPC_PORT` | 8030 | gRPC 서버 포트 |
| `AWS_PROFILE` | tacos | AWS 프로필 (필수) |
| `AWS_REGION` | ap-northeast-2 | AWS 리전 |
| `EVENT_BUS_NAME` | ticket-reservation-events | EventBridge 버스 이름 |
| `PAYMENT_EVENT_SOURCE` | payment-sim-api | 이벤트 소스 식별자 |
| `PAYMENT_WEBHOOK_QUEUE_URL` | - | SQS 큐 URL (필수) |
| `PAYMENT_WEBHOOK_DLQ_URL` | - | SQS DLQ URL (선택) |
| `WEBHOOK_SECRET` | payment-sim-dev-secret | Webhook 서명 시크릿 |
| `DEFAULT_SCENARIO` | approve | 기본 결제 시나리오 |
| `DEFAULT_DELAY_MS` | 2000 | 지연 시나리오 시간 |

### 포트 정보
- **8030**: gRPC 서비스
- **8031**: 헬스체크 + 메트릭스 (HTTP)

## 🔗 서비스 연동

### 비동기 이벤트 플로우

```
1. Client → Payment-Sim-API (결제 요청)
2. Payment-Sim-API → EventBridge (이벤트 발행)
3. EventBridge → SQS (이벤트 라우팅)
4. Reservation-Worker → SQS (메시지 소비)
5. Payment-Sim-API → HTTP Webhook (병행 처리)
```

### 서비스 간 통신 플로우

```
Gateway API (8000) → (gRPC) → Payment Sim API (8030)
Reservation API (8010) → (gRPC) → Payment Sim API (8030)
Payment Sim API (8030) → (EventBridge + Webhook) → Reservation API (8010)
```

### 🎯 실제 테스트 방법

#### 1. grpcui 웹 인터페이스 (추천)
```bash
# grpcui 실행
grpcui -plaintext localhost:8030

# 브라우저에서 나타나는 URL로 접속
# → payment.v1.PaymentService 선택
# → CreatePaymentIntent 선택 후 아래 데이터 입력:
```

**테스트 데이터 예시:**
```json
{
  "reservation_id": "res-test-123",
  "user_id": "user-test-456",
  "amount": {
    "amount": 100000,
    "currency": "KRW"
  },
  "scenario": "approve",
  "webhook_url": "https://httpbin.org/post"
}
```

#### 2. grpcurl 명령줄 테스트
```bash
# 서비스 목록 확인
grpcurl -plaintext localhost:8030 list

# 결제 Intent 생성
grpcurl -plaintext -d '{
  "reservation_id": "res_123",
  "user_id": "user_456",
  "amount": {"amount": 100000, "currency": "KRW"},
  "scenario": "approve",
  "webhook_url": "https://httpbin.org/post"
}' localhost:8030 payment.v1.PaymentService/CreatePaymentIntent
```

**예상 결과:**
1. ✅ **즉시 응답**: PENDING 상태
2. ✅ **2초 후**: EventBridge 이벤트 발송
3. ✅ **동시에**: HTTP Webhook 전송
4. ✅ **Reservation Worker**: SQS에서 메시지 소비

## 🎯 AWS 인프라 설정

### EventBridge 룰 생성
```bash
# EventBridge 룰 생성
aws events put-rule --profile tacos --region ap-northeast-2 \
  --name "payment-events-to-sqs" \
  --event-pattern '{"source": ["payment-sim-api"], "detail-type": ["Payment Status Updated"]}' \
  --event-bus-name "ticket-reservation-events"

# SQS 타겟 추가
aws events put-targets --profile tacos --region ap-northeast-2 \
  --rule "payment-events-to-sqs" \
  --event-bus-name "ticket-reservation-events" \
  --targets 'Id=1,Arn=arn:aws:sqs:ap-northeast-2:YOUR_ACCOUNT_ID:traffic-tacos-payment-webhooks'
```

### Reservation Worker 실행
```bash
# 환경 변수 설정하여 실행
PAYMENT_WEBHOOK_QUEUE_URL="https://sqs.ap-northeast-2.amazonaws.com/YOUR_ACCOUNT_ID/traffic-tacos-payment-webhooks" \
AWS_PROFILE=tacos \
AWS_REGION=ap-northeast-2 \
./bin/reservation-worker
```

## 🎯 성능 목표

- **처리량**: 30k RPS 시스템 일부
- **지연시간**: P95 < 50ms (결제 시뮬레이션)
- **가용성**: 99.9% 목표
- **Webhook 지연**: 기본 2초 (설정 가능)

## 🏷️ MSA 표준 준수

- ✅ **Dockerfile**: Multi-stage 빌드
- ✅ **Makefile**: 표준 CI/CD 명령어
- ✅ **순수 gRPC**: inventory-api 패턴 따름
- ✅ **grpcui**: gRPC 웹 인터페이스 지원
- ✅ **Prometheus**: 메트릭스 수집 (/metrics)
- ✅ **Kubernetes**: 헬스체크 지원 (/health)
- ✅ **AWS Profile**: 'tacos' 프로필 사용
- ✅ **환경변수**: 로컬 .env 파일 지원
- ✅ **Event-Driven**: EventBridge + SQS 비동기 처리

## 🧪 테스트

### 유닛 테스트
```bash
go test ./...
```

### 통합 테스트
```bash
# 1. 서비스 헬스체크
curl http://localhost:8031/health

# 2. gRPC 서비스 확인
grpcurl -plaintext localhost:8030 list

# 3. gRPC 웹 인터페이스 테스트
grpcui -plaintext localhost:8030

# 4. 실제 결제 플로우 테스트
grpcurl -plaintext -d '{"reservation_id":"test-123","user_id":"user-456","amount":{"amount":100000,"currency":"KRW"},"scenario":"approve","webhook_url":"https://httpbin.org/post"}' localhost:8030 payment.v1.PaymentService/CreatePaymentIntent
```

### 성능 테스트
```bash
# gRPC 성능 테스트 (ghz 설치 필요)
go install github.com/bojand/ghz/cmd/ghz@latest

# 결제 생성 성능 테스트
ghz --insecure --proto proto/payment/v1/payment.proto \
  --call payment.v1.PaymentService.CreatePaymentIntent \
  -d '{"reservation_id":"perf-test","user_id":"user-test","amount":{"amount":100000,"currency":"KRW"},"scenario":"approve"}' \
  -c 10 -n 1000 localhost:8030

# 부하 테스트 (30초간)
ghz --insecure --proto proto/payment/v1/payment.proto \
  --call payment.v1.PaymentService.CreatePaymentIntent \
  -d '{"reservation_id":"load-test","user_id":"user-load","amount":{"amount":50000,"currency":"KRW"},"scenario":"random"}' \
  -c 50 -z 30s localhost:8030
```

## 🔍 모니터링

### 헬스체크
- **HTTP**: `GET :8031/health`
- **Kubernetes**: 포트 8031로 헬스체크 설정
- **Docker**: `HEALTHCHECK` 자동 설정

### 로깅
- **개발**: 컬러 출력, DEBUG 레벨
- **운영**: JSON 형태, INFO 레벨
- **트레이스 ID**: 분산 추적 지원

## 📦 배포

### 독립 배포
각 서비스는 독립적으로 배포 가능:

```bash
# Docker 배포
docker build -t payment-sim-api:v1.0.0 .
docker push payment-sim-api:v1.0.0

# Kubernetes 배포 (예시)
kubectl apply -f k8s/payment-sim-api.yaml
```

## 🤝 기여하기

1. 개발 환경 설정: `./scripts/run_local.sh setup`
2. 변경사항 적용
3. 테스트 실행: `make ci`
4. PR 생성

## 📞 지원

- **이슈 리포팅**: GitHub Issues
- **아키텍처 문의**: MSA 팀
- **운영 지원**: DevOps 팀