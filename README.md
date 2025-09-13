# Payment Simulator API

Traffic Tacos 플랫폼의 결제 모의 서비스입니다. 예약 시스템의 결제 단계를 시뮬레이션하고 웹훅 콜백을 제공합니다.

## 🚀 기능

- **결제 인텐트 생성**: 다양한 시나리오(승인/실패/지연/랜덤) 기반 결제 시뮬레이션
- **웹훅 콜백**: HMAC 서명된 안전한 웹훅 전송
- **멱등성 보장**: 동일 요청에 대한 일관된 응답
- **관측 가능성**: OpenTelemetry 트레이싱, Prometheus 메트릭스, 구조화된 로깅
- **고성능**: 수천 RPS 처리 가능

## 📋 사전 요구사항

- Go 1.22+
- Docker (선택사항)

## 🏃‍♂️ 로컬 실행

### 방법 1: 스크립트 사용
```bash
./scripts/run_local.sh
```

### 방법 2: 직접 실행
```bash
# 빌드
make build

# 실행
WEBHOOK_SECRET=your-secret-key ./bin/payment-sim-api
```

### 방법 3: Docker 사용
```bash
# 빌드 및 실행
make docker-build
make docker-run
```

## 📚 API 문서

- **API 문서**: http://localhost:8080 (실행 시)
- **OpenAPI 스펙**: `openapi/payment-sim.yaml`
- **메트릭스**: http://localhost:8080/metrics
- **헬스 체크**: http://localhost:8080/healthz

## 🧪 테스트

```bash
# 모든 테스트 실행
make test

# 단위 테스트만
make test-unit

# 커버리지 보고서
make test-coverage

# 통합 테스트
make test-integration
```

## 🔧 설정

환경변수를 통해 설정을 변경할 수 있습니다:

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `PORT` | `8080` | 서버 포트 |
| `WEBHOOK_SECRET` | 필수 | 웹훅 HMAC 서명 키 |
| `LOG_LEVEL` | `info` | 로깅 레벨 |
| `DEFAULT_APPROVE_DELAY_MS` | `200` | 승인 시나리오 지연 시간 |
| `DEFAULT_FAIL_DELAY_MS` | `100` | 실패 시나리오 지연 시간 |
| `DEFAULT_DELAY_DELAY_MS` | `3000` | 지연 시나리오 기본 시간 |
| `RANDOM_APPROVE_RATE` | `0.8` | 랜덤 시나리오 승인 확률 |
| `WEBHOOK_TIMEOUT_MS` | `1000` | 웹훅 요청 타임아웃 |
| `WEBHOOK_MAX_RETRIES` | `5` | 웹훅 최대 재시도 횟수 |
| `WEBHOOK_BACKOFF_MS` | `1000` | 재시도 백오프 기본 시간 |
| `WEBHOOK_MAX_RPS` | `500` | 웹훅 최대 RPS (0: 무제한) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://otel-collector:4317` | OpenTelemetry 엔드포인트 |

## 📊 API 사용 예제

### 결제 인텐트 생성
```bash
curl -X POST http://localhost:8080/v1/sim/intent \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
    "reservation_id": "rsv_abc123",
    "amount": 120000,
    "scenario": "approve",
    "webhook_url": "https://your-webhook-endpoint.com/callback"
  }'
```

### 응답
```json
{
  "payment_intent_id": "pay_01K51CTDK6MJY73APY8BEJKQDA",
  "status": "APPROVED",
  "next": "webhook"
}
```

### 결제 인텐트 조회
```bash
curl http://localhost:8080/v1/sim/intents/pay_01K51CTDK6MJY73APY8BEJKQDA
```

## 🏗️ 아키텍처

```
payment-sim-api/
├── cmd/payment-sim-api/     # 애플리케이션 엔트리포인트
├── internal/
│   ├── config/             # 설정 관리
│   ├── http/               # HTTP 서버, 핸들러, 미들웨어
│   ├── service/            # 비즈니스 로직
│   ├── store/              # 데이터 저장소 (인메모리)
│   ├── webhook/            # 웹훅 디스패처
│   └── observability/      # 로깅, 메트릭스, 트레이싱
├── openapi/                # API 문서
├── scripts/                # 유틸리티 스크립트
└── test/                   # 테스트 코드
```

## 🔒 보안

- **HMAC 서명**: 모든 웹훅에 SHA256 HMAC 서명
- **멱등성**: Idempotency-Key 헤더로 중복 요청 방지
- **타임아웃**: 모든 외부 요청에 타임아웃 적용
- **재시도**: 지수 백오프로 웹훅 재전송

## 📈 성능

- **P95 지연시간**: < 20ms (웹훅 전송 제외)
- **동시성**: 고루틴 기반 비동기 처리
- **메모리**: 인메모리 저장소로 빠른 응답
- **확장성**: 설정 가능한 워커 풀과 큐

## 🤝 기여

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 라이선스

이 프로젝트는 MIT 라이선스를 따릅니다.

## 📞 연락처

Traffic Tacos Team - dev@traffic-tacos.com
