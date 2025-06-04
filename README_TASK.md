# Mini-S3 Local Object Store (Go)

## 0 · Context & Objective
Build a minimalist, self‑hosted object store that speaks a **subset** of Amazon S3’s path‑style API.  
Your service **must**:

* Stream large files to disk with constant memory usage.  
* Serve byte‑range reads efficiently.  
* Shut down gracefully while protecting in‑flight uploads.

You have **90 minutes of coding time** (followed by a 30‑minute walkthrough).

---

## 1 · Environment

* **Language** Go ≥ 1.22 (Standard library is sufficient; third‑party modules allowed if justified in the README).  
* **Starter repo** Provided at interview start.

```
mini-s3/
├── main.go              // boots an empty HTTP server
├── store/
│   └── store.go         // interface + TODO stubs
├── internal/
│   └── hash.go          // StreamSHA256 helper
├── test/
│   └── grader.sh        // curl-based smoke tests
├── go.mod               // `go mod init` already done
└── README_TASK.md       // condensed rules + cheat‑sheet
```

The repo **compiles and tests green** before you start; your edits must not break  
`go vet ./...` **or** `go test ./...`.

---

## 2 · Tasks & Requirements

### 2.1  Mandatory · **Foundation (Basics)**  
_Complete **all five** – the grader hard‑fails on any miss._

1. **PUT** `/<bucket>/<key>`  
   * Stream request body → `./data/<bucket>/<key>` (create parent dirs as needed).  
   * Respond `200 OK` with JSON `{ "etag": "<hex‑sha256>" }`.

2. **GET** `/<bucket>/<key>`  
   * Stream file back.  
   * Return `Content‑Length`, `ETag`, and `Last‑Modified` headers.

3. **HEAD** `/<bucket>/<key>`  
   * Same headers as GET, zero body.

4. **DELETE** `/<bucket>/<key>`  
   * Remove file.  
   * Return `204 No Content`.

5. **Robust error handling**  
   * `404` – bucket/key not found.  
   * `400` – empty bucket or key segment.  
   * Never crash on short reads, permission errors, or partial deletes.

---

### 2.2  Mandatory · **Production‑Grade (Medium)**  
_Do all three._

6. **Constant‑memory streaming**  
   * Uploading a **250 MB** file must keep RSS **< 40 MB**.  
   * Hidden tests sample `/proc/$PID/status`.

7. **Single‑range support**  
   * Handle `Range: bytes=<start>-<end>`; return `206 Partial Content` and correct `Content‑Range`.

8. **Graceful shutdown**  
   * Trap `SIGINT`; stop accepting new connections, finish in‑flight transfers, flush, exit within **3 s**.

---

### 2.3  Optional · **Advanced Extensions**  
_Pick **any two** (extra credit for more)._

| ID | Extension | Acceptance criteria |
|----|-----------|---------------------|
| **A** | **Prometheus metrics** | `/metrics` exposes op counts, total bytes, p95 latency. |
| **B** | **Presigned URL generator** | `GET /sign?bucket=…&key=…&expires=60` returns one‑time URL secured with HMAC‑SHA256 of server secret. |
| **C** | **Simple multipart upload** | 3‑call flow: `POST /multipart/init`, `PUT /multipart/<id>/<partNo>`, `POST /multipart/complete`. Concatenate parts; reject out‑of‑order. |
| **D** | **Bucket disk‑quota** | CLI flag `--quota-mb=500`; PUT beyond quota returns `413 Payload Too Large`. |
| **E** | **Embedded Web UI** | Serve `/ui` with minimal SPA that lists objects and supports upload/download via fetch‑stream. |

---

## 3 · Operational Constraints

* **Concurrency** Grader fires **8 parallel PUTs**; corruption or race‑detector failures are fatal.  
* **Throughput** Single 250 MB PUT must complete in **≤ 8 s** (grader box).  
* **Filesystem root** All data stays under `./data/`; no writes elsewhere.  
* **Third‑party libs** Allowed if you pin versions and justify.  
* **Tests** Add at least **one** happy‑path and **one** edge‑case test (`go test ./...` must pass).

---

## 4 · Submission Checklist

1. Commit: `git commit -m "Task complete"` – no lingering TODOs in mandatory sections.  
2. Update **CHANGELOG.md** with completed features & known gaps.  
3. Run `./test/grader.sh` – all checks green.  
4. Push branch or deliver ZIP; we re‑run hidden load + race + RSS tests.

---

## 5 · Scoring Rubric (100 pts)

| Area | Pts | Notes |
|------|-----|-------|
| Correctness & API spec | 30 | All mandatory endpoints behave as defined. |
| Streaming efficiency | 15 | Pass RSS + throughput gates. |
| Concurrency safety | 15 | `go test -race` clean; no lost writes. |
| Graceful ops | 10 | Clean SIGINT exit, proper logs. |
| Code quality & tests | 15 | Idiomatic structuring, clear error wraps, ≥ 2 tests. |
| Optional extensions | 10 | 5 pts each, up to 10. |
| Communication | 5 | README clarity, trade‑off notes, concise commits. |

Score **≥ 70** → pass · **≥ 85** → strong · **90+** → exceptional.

---

### FAQ

**Q : May I use framework X?** Yes, if you justify the dependency and meet all constraints.  
**Q : Must I implement multipart?** Only if you pick optional task C.  
**Q : Can I tweak buffer sizes?** Absolutely—document your rationale.  
**Q : Is authentication required?** No; optional task B covers signing.

Good luck — ship something you’d be happy to run in prod!  
