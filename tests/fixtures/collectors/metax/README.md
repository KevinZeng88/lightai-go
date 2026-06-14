# MetaX C500 8-GPU fixtures

These fixtures were captured from a real MetaX C500 server.

Environment:
- Hostname: k8s-master1
- GPU: 8 x MetaX C500
- Agent metrics endpoint: http://127.0.0.1:19091/metrics
- Observed issue: Agent /metrics returned HTTP 500 because duplicate Prometheus time series were emitted.

Files:
- discover_8x_c500.txt: raw output from deploy/collectors/gpu/metax/discover.sh
- metrics_8x_c500.txt: raw output from deploy/collectors/gpu/metax/metrics.sh
- agent_metrics_duplicate_error_8x_c500.txt: raw /metrics error output showing duplicate lightai_gpu_available_status series

Important characteristics:
- discover output may contain memory_total_bytes=null.
- metrics output contains authoritative memory_total_bytes, memory_used_bytes, memory_free_bytes.
- 8 GPUs must remain 8 GPUs after parsing, merging, repeated collection, and Prometheus export.
- The exporter must not emit duplicate metric name + label set in one scrape.
