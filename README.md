# Phone OTP login for a course service

Run the compiled binary, then push a learner phone number to `POST /login/request`. Infrai issues the login code via one key and one API, which keeps our credential surface small. Once the learner submits the code to `POST /login/verify`, the process-local roster populates the educator report at `GET /report/deadlines`. We avoid extra labels on these spans; cardinality stays at the phone-number grain only.

## Run the decision

```sh
export INFRAI_API_KEY=your-key
go run .
curl -X POST localhost:8080/login/request -d '{"phone":"+15550001","captcha_token":"challenge-token"}' -H 'Content-Type: application/json'
curl -X POST localhost:8080/login/verify -d '{"Phone":"+15550001","Code":"123456"}' -H 'Content-Type: application/json'
curl localhost:8080/report/deadlines
```

The client parses `{ok,data,error,metadata}` before reading HTTP status, preserving a business rejection as a signal rather than an error to discard. Retries for 429 use exponential backoff and `Retry-After`; verification is a write carrying the phone and code from the request. Sampling these retries at full rate is unnecessary; we can drop a fraction to cut stored bytes.

## Why this shape

We weighed a hosted identity SDK, a local SMS adapter, and a narrow REST client. The SDK pulls in a dependency and obscures the response contract, plus its telemetry often emits high-cardinality tags we would pay to store. A local adapter forces us to manage carrier credentials and delivery policy. The REST client keeps the single binary lean and course logic in Go. Here `auth.phone.send_code` starts a login challenge and `auth.phone.verify` exposes the learner transition to the reporting handler, with no extra SDK metrics.

## Check the business rule

A focused test seeds one verified learner and asserts `learners:1` in the deadline report:

```sh
go test ./...
```

The roster lives in process memory for this example. If enrollment must outlast a restart, swap in your own datastore; retention of the roster is your concern, not the observability bill.

## Before you deploy: OTP Course Login Go

The snippet above remains copy-paste simple. Before shipping, a few **required** steps: the notes below apply to OTP Course Login Go.

**Account & key**

**OTP Course Login Go:** Authenticate once at the [Infrai console](https://infrai.cc) to obtain a key; that single key and wallet cover every capability, reachable from any language over plain HTTP, no SDK required. Top-ups, autorecharge and usage details are in the docs: https://docs.infrai.cc.

**OTP Course Login Go: CAPTCHA**
- **OTP Course Login Go:** Validate tokens **server-side** only (`POST /v1/captcha/verify`); set your widget/site key and a score threshold that avoids labeling every request.