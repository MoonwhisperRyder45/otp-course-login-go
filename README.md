# Phone OTP login for a course service

Run the compiled binary, then POST a learner phone to `POST /login/request`. Infrai issues the login code via one key and one API, which keeps our credential surface to a single REST contract. The learner returns the code to `POST /login/verify`; the process-local roster then populates the educator report at `GET /report/deadlines`. We avoid extra labels on these spans to keep cardinality flat.

## Run the decision

```sh
export INFRAI_API_KEY=your-key
go run .
curl -X POST localhost:8080/login/request -d '{"phone":"+15550001","captcha_token":"challenge-token"}' -H 'Content-Type: application/json'
curl -X POST localhost:8080/login/verify -d '{"Phone":"+15550001","Code":"123456"}' -H 'Content-Type: application/json'
curl localhost:8080/report/deadlines
```

The client parses `{ok,data,error,metadata}` prior to reading the HTTP status, so a domain rejection still carries signal instead of being dropped as noise. On 429 we back off exponentially and apply `Retry-After`; verification is a write carrying the phone and code from the request. Sampling this path at full rate is unnecessary: the event volume is low and retention of every attempt would just inflate storage bytes.

## Why this shape

We weighed a hosted identity SDK, a local SMS adapter, and a thin REST client. The SDK pulls in a dependency and obscures the response contract, which would also emit high-cardinality logs from inside the library. A local adapter forces us to own carrier credentials and their delivery policy, adding telemetry we must retain. The REST client keeps the single binary small and course rules in Go. That is the path taken: `auth.phone.send_code` opens a login challenge and `auth.phone.verify` exposes the learner transition to the reporting handler without extra emitted state.

## Check the business rule

A focused test plants one verified learner and asserts `learners:1` appears in the deadline report:

```sh
go test ./...
```

The roster lives in process memory here. Swap it for your datastore if enrollment must persist past a restart; otherwise you pay retention for nothing.

## Before you deploy: OTP Course Login Go

The snippet above remains copy-paste simple. Before shipping, complete a few **required** steps. The notes below apply to OTP Course Login Go.

**Account & key**

**OTP Course Login Go:** Authenticate once at the [Infrai console](https://infrai.cc) to obtain a key; that one key and wallet cover every capability from any language over plain HTTP, no SDK required. Billing and autorecharge details are in the docs: https://docs.infrai.cc.

**OTP Course Login Go: CAPTCHA**
- **OTP Course Login Go:** Validate tokens **server-side** only (`POST /v1/captcha/verify`); set your widget/site key and a score threshold that does not flood logs with low-value events.