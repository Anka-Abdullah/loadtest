# Penjelasan Detail Penggunaan Load Test

## 1. Basic Load Test

```bash
./loadtest -n 1000 -c 100 https://jsonplaceholder.typicode.com/posts
```

### Parameter Breakdown:
- `-n 1000` → Jumlah total request yang akan dikirim
- `-c 100` → Concurrent users/connections (100 users secara bersamaan)
- `URL` → Target endpoint yang akan di-test

### Apa yang terjadi:
- Aplikasi membuat 100 worker/goroutine (karena `-c 100`)
- Setiap worker akan mengirim request ke endpoint
- Total 1000 request akan dikirim secara paralel
- Jika ada 100 workers dan 1000 requests, setiap worker akan mengirim ~10 requests

### Use Case:
- Test kapasitas server menangani traffic
- Cek stability dengan 100 concurrent connections
- Monitor response time under load

### Expected Output:

```text
🚀 Memulai load test...
   URL: https://jsonplaceholder.typicode.com/posts
   Requests: 1000
   Concurrency: 100
   Method: GET

📊 Menjalankan requests...
   Progress: 100/1000 requests
   Progress: 200/1000 requests
   ...

==================================================
📈 HASIL LOAD TEST
==================================================
Total waktu:           12.34s
Total requests:        1000
Requests sukses:       995
Requests gagal:        5
Requests per detik:    81.03
Rata-rata latency:     1.23s
Latency terendah:      456ms
Latency tertinggi:     3.45s

📊 Status Codes:
  200: 995 (99.5%)
  500: 5 (0.5%)
```

## 2. POST Request dengan JSON Payload

```bash
./loadtest -n 500 -c 50 -m POST -d '{"title":"test"}' https://jsonplaceholder.typicode.com/posts
```

### Parameter Tambahan:
- `-m POST` → HTTP method POST (default: GET)
- `-d '{"title":"test"}'` → Request body (JSON payload)

### Detail Teknis:

Headers otomatis:
```text
Content-Type: application/json
Content-Length: 18
User-Agent: Go-Load-Tester/1.24
```

Body yang dikirim:
```json
{"title":"test"}
```

### Flow:
- 50 concurrent workers
- Setiap worker mengirim POST request dengan payload JSON
- Total 500 requests dikirim

### Use Case:
- Test API yang menerima data (create/update)
- Cek performance database write operations
- Validasi JSON parsing capacity

### Expected Behavior:

```json
// Response dari jsonplaceholder:
{
  "id": 101,
  "title": "test",
  "body": "test body",
  "userId": 1
}
```

## 3. Dengan Custom Headers

```bash
./loadtest -n 200 -c 20 -H "Authorization: Bearer test" https://httpbin.org/headers
```

### Parameter:
- `-H "Authorization: Bearer test"` → Custom HTTP header
- `https://httpbin.org/headers` → Endpoint khusus yang mengembalikan headers yang diterima

### Apa yang terjadi:

Setiap request akan memiliki header:
```text
Authorization: Bearer test
User-Agent: Go-Load-Tester/1.24
Accept: */*
```

httpbin.org/headers akan mengembalikan semua headers yang diterima:
```json
{
  "headers": {
    "Accept": "*/*", 
    "Authorization": "Bearer test",
    "Host": "httpbin.org", 
    "User-Agent": "Go-Load-Tester/1.24"
  }
}
```

### Use Case:
- Test authentication/authorization endpoints
- Validasi header parsing di server
- Simulasi API dengan token-based auth

## 4. Kombinasi Lengkap Contoh Lain

### A. Test dengan Timeout

```bash
./loadtest -n 300 -c 30 -t 10 https://httpbin.org/delay/5
```

- `-t 10` → Timeout 10 detik per request
- Endpoint `/delay/5` → Server delay 5 detik sebelum response
- **Use:** Test handling slow responses

### B. Multiple Headers

```bash
./loadtest -n 100 -c 10 \
  -H "Authorization: Bearer token123" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: myapikey" \
  -m PUT \
  -d '{"status":"active"}' \
  https://api.example.com/v1/users/1
```

### C. Test Error Rate

```bash
./loadtest -n 1000 -c 100 https://httpbin.org/status/500
```

- 500 Internal Server Error → Test error handling
- Monitor berapa % request yang gagal

### D. Test dengan Random Delay (Simulasi Real Traffic)

```bash
# Sequential test dengan delay berbeda
./loadtest -n 50 -c 5 https://httpbin.org/delay/1
./loadtest -n 50 -c 5 https://httpbin.org/delay/3
./loadtest -n 50 -c 5 https://httpbin.org/delay/5
```

## 5. Interpretasi Hasil & Metrics

### Metrics Penting:

#### Requests per Second (RPS)
- **81.03 requests/detik**
- **Interpretasi:** Server mampu handle 81 request/detik dengan config saat ini

#### Success Rate
- **995/1000 = 99.5%**
- **Benchmark:**
  - ≥ 99% → Excellent
  - 95-99% → Good
  - 90-95% → Warning
  - < 90% → Problematic

#### Latency Distribution:
```text
Avg: 1.23s
Min: 456ms
Max: 3.45s
P50/P95/P99 (jika ada): Lebih baik untuk analisis
```

#### Status Codes:
- **200 OK** → Success
- **500 Internal Server Error** → Server error
- **429 Too Many Requests** → Rate limiting
- **502 Bad Gateway** → Upstream issues

## 6. Best Practices Testing

### Fase 1: Baseline Test

```bash
# Test ringan untuk baseline
./loadtest -n 100 -c 10 https://api.example.com/health
```

### Fase 2: Ramp-up Test

```bash
# Tingkatkan secara bertahap
./loadtest -n 500 -c 50 https://api.example.com/api
./loadtest -n 2000 -c 200 https://api.example.com/api
./loadtest -n 5000 -c 500 https://api.example.com/api
```

### Fase 3: Stress Test

```bash
# Max capacity test
./loadtest -n 10000 -c 1000 https://api.example.com/api
```

### Fase 4: Endurance Test

```bash
# Test dalam waktu lama
./loadtest -n 50000 -c 100 https://api.example.com/api

# atau dengan loop
for i in {1..10}; do
  ./loadtest -n 5000 -c 100 https://api.example.com/api
  sleep 10
done
```

## 7. Contoh Real-World Scenario

### E-commerce API Test:

```bash
# 1. Test product listing (GET, high traffic)
./loadtest -n 5000 -c 200 https://api.store.com/products

# 2. Test product detail (GET, medium traffic)
./loadtest -n 2000 -c 100 https://api.store.com/products/123

# 3. Test add to cart (POST, authenticated)
./loadtest -n 1000 -c 50 -m POST \
  -H "Authorization: Bearer user_token" \
  -d '{"product_id":123,"quantity":1}' \
  https://api.store.com/cart

# 4. Test checkout (POST, complex operation)
./loadtest -n 500 -c 20 -m POST \
  -H "Authorization: Bearer user_token" \
  -d '{"cart_id":"abc123","payment_method":"credit_card"}' \
  https://api.store.com/checkout
```

### Monitoring selama test:

```bash
# Jalankan test dan simpan hasil ke file
./loadtest -n 1000 -c 100 https://api.example.com/endpoint > hasil-test.log

# Monitor dengan tail
tail -f hasil-test.log

# Atau dengan tee untuk real-time monitoring
./loadtest -n 1000 -c 100 https://api.example.com/endpoint | tee hasil-test.log
```

## 8. Troubleshooting Tips

### Jika banyak error:

```bash
# Kurangi concurrency
./loadtest -n 1000 -c 10  # dari 100 ke 10

# Tambah timeout
./loadtest -n 1000 -c 100 -t 60

# Test endpoint sederhana dulu
./loadtest -n 100 -c 10 https://httpbin.org/get
```

### Jika latency tinggi:

Cek network latency:
```bash
ping api.example.com
curl -o /dev/null -s -w 'Total: %{time_total}s\n' https://api.example.com
```

Bandingkan dengan baseline:
```bash
# Single request
time curl https://api.example.com/api

# VS load test
./loadtest -n 10 -c 1 https://api.example.com/api
```

## 9. Contoh Output Lengkap dengan Analisis

```text
🚀 Memulai load test...
   URL: https://jsonplaceholder.typicode.com/posts
   Requests: 1000
   Concurrency: 100
   Method: GET

📊 Menjalankan requests...
   Progress: 100/1000 requests ✓
   Progress: 200/1000 requests ✓
   Progress: 300/1000 requests ✓
   Progress: 400/1000 requests ✓
   Progress: 500/1000 requests ✓
   ❌ Request 501 gagal: timeout
   ❌ Request 502 gagal: timeout
   Progress: 600/1000 requests ✓
   ...

==================================================
📈 HASIL LOAD TEST
==================================================
Total waktu:           15.67s
Total requests:        1000
Requests sukses:       985
Requests gagal:        15
Requests per detik:    63.82  ⚠️ (di bawah target 100 RPS)
Rata-rata latency:     1.56s  ⚠️ (terlalu tinggi untuk GET sederhana)
Latency terendah:      234ms
Latency tertinggi:     4.89s  ⚠️ (outlier sangat tinggi)

📊 Status Codes:
  200: 985 (98.5%)
  500: 10 (1.0%)
  502: 5 (0.5%)

==================================================
⚠️  Load test PERINGATAN (Success rate ≥ 80%)
==================================================

ANALYSIS:
1. 1.5% failure rate - Perlu investigasi timeout
2. Avg latency 1.56s - Kemungkinan server overload atau network issue
3. Max latency 4.89s - Ada beberapa request yang sangat lambat
4. 502 errors - Indikasi upstream/proxy issues

RECOMMENDATION:
1. Cek server resources (CPU, memory)
2. Review database connection pool
3. Implement caching untuk endpoint ini
4. Monitor network latency antara client-server
```