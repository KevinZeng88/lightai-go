| Backend | Template copy | NBR check | Runtime edit | RunPlan preview | Docker preview | Ports | Device binding | Detail/list/edit | Runtime start |
| -- | -- | -- | -- | -- | -- | -- | -- | -- | -- |
| vllm | PASS | ready | PASS | PASS | PASS | host 18081 / container 8000 / health 18081 | PASS | PASS | dry-run/API evidence |
| sglang | PASS | ready | PASS | PASS | PASS | host 18082 / container 30000 / health 18082 | PASS | PASS | dry-run/API evidence |
| llamacpp | PASS | ready | PASS | PASS | PASS | host 18083 / container 8080 / health 18083 | PASS | PASS | PASS: running |

Device binding source: placement_json.accelerator_ids selects the node GPU; resolver maps GPU id to NVIDIA index 0, producing Docker `--gpus device=0` and `CUDA_VISIBLE_DEVICES=0` in the resolved RunPlan/command preview.
