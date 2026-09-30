# Benchmark Results: pure-go-docx

Measured on: **2026-09-30**  
Environment:
- **OS / Arch**: `linux/amd64`
- **CPU**: `11th Gen Intel(R) Core(TM) i7-1165G7 @ 2.80GHz`
- **Command**: `make bench` (`go test -run '^$' -bench '.' -benchmem -count 1 ./...`)

---

## 1. Fixture Benchmarks

Measures opening, parsing, text/markdown rendering, RAG chunking, and lazy image loading on test fixtures:
- `service-agreement-en.docx` (generated clean fixture with headers, footers, footnotes)
- `service-contract-sample.docx` (real-world Word-authored contract with styles, tables, and media)
- `images.docx` (embedded images test fixture)

| Benchmark | Fixture / Target | Time / op | Throughput | Memory / op | Allocs / op |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`BenchmarkOpenReader`** | `service-agreement` | **647.7 µs** (647 694 ns) | 12.77 MB/s | 267.1 KB (267 144 B) | 5 384 |
| **`BenchmarkOpenReader`** | `service-contract-real` | **5.87 ms** (5 868 416 ns) | 9.37 MB/s | 2.34 MB (2 337 250 B) | 51 523 |
| **`BenchmarkOpen`** (disk) | `service-agreement` | **750.9 µs** (750 874 ns) | 11.01 MB/s | 286.6 KB (286 626 B) | 5 398 |
| **`BenchmarkOpen`** (disk) | `service-contract-real` | **5.82 ms** (5 824 020 ns) | 9.44 MB/s | 2.47 MB (2 469 121 B) | 51 544 |
| **`BenchmarkToText`** | `service-agreement` | **9.18 µs** (9 175 ns) | — | 13.8 KB (13 800 B) | 127 |
| **`BenchmarkToText`** | `service-contract-real` | **87.1 µs** (87 131 ns) | — | 306.9 KB (306 906 B) | 430 |
| **`BenchmarkToMarkdown`** | `service-agreement` | **15.8 µs** (15 769 ns) | — | 21.8 KB (21 832 B) | 130 |
| **`BenchmarkToMarkdown`** | `service-contract-real` | **144.2 µs** (144 166 ns) | — | 372.6 KB (372 636 B) | 438 |
| **`BenchmarkChunks`** | `service-agreement` | **46.7 µs** (46 730 ns) | — | 48.1 KB (48 136 B) | 768 |
| **`BenchmarkChunks`** | `service-contract-real` | **264.0 µs** (263 951 ns) | — | 385.8 KB (385 786 B) | 2 130 |
| **`BenchmarkAllImages`** | `images` | **15.7 µs** (15 691 ns) | — | 4.75 KB (4 754 B) | 44 |
| **`BenchmarkAllImages`** | `service-agreement-en` | **2.02 µs** (2 018 ns) | — | 1.90 KB (1 901 B) | 21 |
| **`BenchmarkAllImages`** | `service-contract-sample` | **5.46 µs** (5 464 ns) | — | 24.0 KB (24 048 B) | 19 |

---

## 2. Scaling Benchmarks (Synthetic Bodies)

Measures linear complexity on synthetic documents with mixed paragraphs, headings, and 3×3 tables at 100, 1 000, and 10 000 blocks.

| Benchmark | Block Count | Time / op | Time Growth | Memory / op | Memory Growth | Allocs / op | Allocs Growth |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`OpenReaderScaling`** | 100 | **0.63 ms** | 1.0× | 312.9 KB | 1.0× | 7 106 | 1.0× |
| | 1 000 | **5.89 ms** | **9.35×** | 2.88 MB | **9.19×** | 68 795 | **9.68×** |
| | 10 000 | **56.92 ms** | **9.67×** | 28.72 MB | **9.98×** | 685 567 | **9.96×** |
| **`ChunksScaling`** | 100 | **0.086 ms** | 1.0× | 97.4 KB | 1.0× | 1 390 | 1.0× |
| | 1 000 | **0.96 ms** | **11.22×** | 949.3 KB | **9.74×** | 14 800 | **10.65×** |
| | 10 000 | **8.68 ms** | **9.02×** | 9.47 MB | **9.97×** | 148 900 | **10.06×** |

---

## 3. Key Findings

1. **Strict $O(N)$ Linearity**:
   - Both parsing and chunking scale strictly linearly with document size.
   - For a 10× increase in block count, execution time and memory increase by ~9.0×–10.0×.
   - A 10 000-block document (~book length with thousands of table cells) parses in **56.9 ms** and chunks in **8.7 ms**.

2. **Parser Throughput & Memory Buffering**:
   - Fast XML tokenization with `Decoder.RawToken` yields ~9.4–12.8 MB/s compressed DOCX throughput.
   - Reading from disk into memory (`Open`) adds negligible overhead compared to in-memory reading (`OpenReader`) on real-world files (~5.82 ms vs ~5.87 ms), while enabling safe lazy image access after file close.

3. **Markdown Streaming Efficiency**:
   - `ToMarkdown` is only ~65% slower than plain text `ToText` (144 µs vs 87 µs on real contracts) with virtually the same allocation count (438 vs 430 allocs).
   - Direct run writing into `strings.Builder` and fast-path text escaping avoid intermediate allocations for formatting tags (`**`, `*`, `<u>`, URLs).

4. **Fast RAG Chunking**:
   - Generating structured RAG chunks for an entire contract takes **0.26 ms**.
   - Capacity pre-estimation (`odhadPoctuChunku`) eliminates slice reallocations during chunk building.

5. **Lazy Image Loading**:
   - On-demand image extraction and decompression take between **2 µs and 15 µs**, keeping initial parsing lightweight when media is not needed.

---

## 4. Raw Output

```text
go test -run '^$' -bench '.' -benchmem -count 1 ./...
goos: linux
goarch: amd64
pkg: github.com/venosm/pure-go-docx
cpu: 11th Gen Intel(R) Core(TM) i7-1165G7 @ 2.80GHz
BenchmarkOpenReader/service-agreement-8         	    1833	    647694 ns/op	  12.77 MB/s	  267144 B/op	    5384 allocs/op
BenchmarkOpenReader/service-contract-real-8     	     199	   5868416 ns/op	   9.37 MB/s	 2337250 B/op	   51523 allocs/op
BenchmarkOpen/service-agreement-8               	    1461	    750874 ns/op	  11.01 MB/s	  286626 B/op	    5398 allocs/op
BenchmarkOpen/service-contract-real-8           	     202	   5824020 ns/op	   9.44 MB/s	 2469121 B/op	   51544 allocs/op
BenchmarkToText/service-agreement-8             	  124934	      9175 ns/op	   13800 B/op	     127 allocs/op
BenchmarkToText/service-contract-real-8         	   13754	     87131 ns/op	  306906 B/op	     430 allocs/op
BenchmarkToMarkdown/service-agreement-8         	   70132	     15769 ns/op	   21832 B/op	     130 allocs/op
BenchmarkToMarkdown/service-contract-real-8     	    8418	    144166 ns/op	  372636 B/op	     438 allocs/op
BenchmarkChunks/service-agreement-8             	   26035	     46730 ns/op	   48136 B/op	     768 allocs/op
BenchmarkChunks/service-contract-real-8         	    4515	    263951 ns/op	  385786 B/op	    2130 allocs/op
BenchmarkAllImages/images-8                     	   76459	     15691 ns/op	    4754 B/op	      44 allocs/op
BenchmarkAllImages/service-agreement-en-8       	  644025	      2018 ns/op	    1901 B/op	      21 allocs/op
BenchmarkAllImages/service-contract-sample-8    	  188253	      5464 ns/op	   24048 B/op	      19 allocs/op
BenchmarkOpenReaderScaling/blocks=100-8         	    1860	    629647 ns/op	   2.54 MB/s	  312904 B/op	    7106 allocs/op
BenchmarkOpenReaderScaling/blocks=1000-8        	     202	   5887839 ns/op	   1.13 MB/s	 2876832 B/op	   68795 allocs/op
BenchmarkOpenReaderScaling/blocks=10000-8       	      19	  56919444 ns/op	   1.01 MB/s	28720532 B/op	  685567 allocs/op
BenchmarkChunksScaling/blocks=100-8             	   13972	     85714 ns/op	   97424 B/op	    1390 allocs/op
BenchmarkChunksScaling/blocks=1000-8            	    1293	    962331 ns/op	  949254 B/op	   14800 allocs/op
BenchmarkChunksScaling/blocks=10000-8           	     136	   8684291 ns/op	 9465510 B/op	  148900 allocs/op
PASS
ok  	github.com/venosm/pure-go-docx	22.457s
```
