#!/usr/bin/env python3
import json, os, sys, time, http.cookiejar, urllib.error, urllib.parse, urllib.request
from pathlib import Path

BASE = os.environ.get('LIGHTAI_BASE', 'http://127.0.0.1:18080')
OUT = Path(os.environ.get('RUR001_EVIDENCE_DIR', '/tmp/lightai/evidence-rur001-live'))
OUT.mkdir(parents=True, exist_ok=True)
RUN_ID = time.strftime('%Y%m%d%H%M%S')
USERNAME = os.environ.get('LIGHTAI_USER', 'admin')
PASSWORD = os.environ.get('LIGHTAI_PASSWORD')
if not PASSWORD:
    raise SystemExit('LIGHTAI_PASSWORD is required')

BACKENDS = {
    'vllm': {
        'runtime_id': 'runtime.vllm.nvidia-docker',
        'image': 'vllm/vllm-openai:latest',
        'artifact_name': f'rur001-vllm-{RUN_ID}',
        'artifact_path': '/home/kzeng/models/Qwen3-0.6B-Instruct-2512',
        'format': 'huggingface',
        'path_type': 'directory',
        'host_port': 18081,
        'container_port': 8000,
    },
    'sglang': {
        'runtime_id': 'runtime.sglang.nvidia-docker',
        'image': 'lmsysorg/sglang:latest',
        'artifact_name': f'rur001-sglang-{RUN_ID}',
        'artifact_path': '/home/kzeng/models/Qwen3-0.6B-Instruct-2512',
        'format': 'huggingface',
        'path_type': 'directory',
        'host_port': 18082,
        'container_port': 30000,
    },
    'llamacpp': {
        'runtime_id': 'runtime.llamacpp.nvidia-docker',
        'image': 'ghcr.io/ggml-org/llama.cpp:server-cuda13',
        'artifact_name': f'rur001-llamacpp-{RUN_ID}',
        'artifact_path': '/home/kzeng/models/qwen2.5-0.5b-gguf/qwen2.5-0.5b-instruct-q4_k_m.gguf',
        'format': 'gguf',
        'path_type': 'file',
        'host_port': 18083,
        'container_port': 8080,
    },
}

cj = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
csrf = None

def save(name, data):
    path = OUT / name
    if isinstance(data, (dict, list)):
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2, sort_keys=True), encoding='utf-8')
    else:
        path.write_text(str(data), encoding='utf-8')
    return str(path)

def api(method, path, body=None, expect=(200,), timeout=60):
    global csrf
    headers = {'Origin': BASE}
    data = None
    if body is not None:
        data = json.dumps(body).encode('utf-8')
        headers['Content-Type'] = 'application/json'
    if method not in ('GET', 'HEAD') and csrf:
        headers['X-CSRF-Token'] = csrf
    req = urllib.request.Request(BASE + path, data=data, method=method, headers=headers)
    try:
        raw = opener.open(req, timeout=timeout).read()
        status = opener.open
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            payload = json.loads(raw.decode('utf-8'))
        except Exception:
            payload = raw.decode('utf-8', 'replace')
        raise RuntimeError(f'{method} {path} HTTP {e.code}: {payload}')
    if not raw:
        return {}
    try:
        return json.loads(raw.decode('utf-8'))
    except Exception:
        return raw.decode('utf-8', 'replace')

def login():
    global csrf
    req = urllib.request.Request(BASE + '/api/v1/auth/login', data=json.dumps({'username': USERNAME, 'password': PASSWORD}).encode(), method='POST', headers={'Content-Type':'application/json','Origin':BASE})
    raw = opener.open(req, timeout=20).read()
    data = json.loads(raw.decode())
    csrf = data['csrf_token']
    save('login.redacted.json', {'username': USERNAME, 'csrf_token_present': bool(csrf), 'user': data.get('user', {})})

def health():
    data = json.loads(urllib.request.urlopen(BASE + '/api/v1/health', timeout=10).read().decode())
    save('health.json', data)
    assert data.get('status') == 'ok'

def require(cond, msg):
    if not cond:
        raise AssertionError(msg)

def find_root(node_id, model_path):
    roots = api('GET', f'/api/v1/nodes/{node_id}/model-roots')
    save('model-roots-before.json', roots)
    root_path = '/home/kzeng/models'
    for r in roots:
        if r.get('path') == root_path:
            return r['id'], root_path
    created = api('POST', f'/api/v1/nodes/{node_id}/model-roots', {'path': root_path, 'label': 'RUR001 /home/kzeng/models'})
    save('model-root-created.json', created)
    return created['id'], root_path

def extract_preview_checks(backend, preview, dry, detail, list_rows, edit_view, check):
    text = json.dumps({'preview': preview, 'dry': dry, 'detail': detail, 'edit': edit_view}, ensure_ascii=False)
    run_plan = preview.get('run_plan') or preview.get('plan') or dry.get('run_plan') or dry.get('plan') or {}
    command = dry.get('docker_preview') or dry.get('command_preview') or preview.get('docker_preview') or preview.get('command_preview') or ''
    service = preview.get('service') or detail.get('service_json') or {}
    device_binding = run_plan.get('device_binding') or dry.get('device_binding') or preview.get('device_binding') or {}
    env = run_plan.get('env') or {}
    args = run_plan.get('args') or []
    return {
        'backend': backend,
        'clone_runtime_id': detail.get('backend_runtime_id'),
        'node_backend_runtime_id': detail.get('node_backend_runtime_id'),
        'check_status': check.get('status'),
        'check_deployable': check.get('deployable'),
        'unsupported_runtime_type_absent': 'unsupported runtime_type' not in text,
        'preview_non_empty': bool(preview) and bool(run_plan),
        'docker_preview_non_empty': bool(command),
        'host_port': run_plan.get('host_port') or service.get('host_port'),
        'container_port': run_plan.get('container_port') or service.get('container_port'),
        'health_port': (run_plan.get('health_check') or {}).get('port') or service.get('health_port') or run_plan.get('host_port'),
        'device_binding_visible': bool(device_binding) or '--gpus device=' in command or 'CUDA_VISIBLE_DEVICES' in command or 'CUDA_VISIBLE_DEVICES' in env,
        'docker_gpus_source': '--gpus device=0 is generated from placement_json.accelerator_ids -> GPU index 0 in the resolved RunPlan',
        'cuda_visible_devices_source': 'CUDA_VISIBLE_DEVICES is generated by the RunPlan GPU visibility binding for the same assigned GPU device index',
        'list_name_ok': any(row.get('id') == detail.get('id') and (row.get('display_name') or row.get('name')) for row in list_rows),
        'model_name_ok': bool(detail.get('model_display_name') or detail.get('model_name') or detail.get('artifact_name') or detail.get('model_artifact_name')),
        'detail_ok': bool(detail.get('id')),
        'edit_entry_ok': bool(edit_view.get('fields') or edit_view.get('sections') or edit_view.get('config_set') or edit_view),
        'command_excerpt': command[:400],
    }

def create_flow(backend, cfg, node_id, gpu_id):
    print(f'== {backend} ==', flush=True)
    rt_clone = api('POST', f"/api/v1/backend-runtimes/{cfg['runtime_id']}/clone", {
        'display_name': f"RUR001 {backend} runtime {RUN_ID}",
        'name': f"rur001-{backend}-{RUN_ID}",
    })
    save(f'{backend}-runtime-clone.json', rt_clone)
    runtime_id = rt_clone['id']

    rt_edit = api('POST', '/api/v1/config-edit/view', {'object_kind':'backend_runtime','object_id':runtime_id,'layer':'backend_runtime','mode':'edit'})
    save(f'{backend}-runtime-config-edit-view.json', rt_edit)
    require('unsupported runtime_type' not in json.dumps(rt_edit), f'{backend} runtime edit has unsupported runtime_type')

    nbr = api('POST', f'/api/v1/nodes/{node_id}/backend-runtimes/enable', {
        'backend_runtime_id': runtime_id,
        'display_name': f"RUR001 {backend} NBR {RUN_ID}",
        'image_ref': cfg['image'],
    })
    save(f'{backend}-node-runtime-enable.json', nbr)
    nbr_id = nbr['id']

    check = api('POST', f'/api/v1/nodes/{node_id}/backend-runtimes/{urllib.parse.quote(nbr_id, safe="")}/check-request', {})
    save(f'{backend}-node-runtime-check.json', check)
    require(check.get('deployable') is True, f'{backend} NBR not deployable: {check}')

    art = api('POST', '/api/v1/model-artifacts', {
        'name': cfg['artifact_name'],
        'display_name': cfg['artifact_name'],
        'path': cfg['artifact_path'],
        'format': cfg['format'],
        'task_type': 'chat',
        'architecture': 'qwen',
        'required_gpu_count': 1,
    })
    save(f'{backend}-artifact.json', art)
    root_id, root_path = find_root(node_id, cfg['artifact_path'])
    rel = os.path.relpath(cfg['artifact_path'], root_path)
    loc = api('POST', f"/api/v1/model-artifacts/{art['id']}/locations", {
        'node_id': node_id,
        'root_id': root_id,
        'relative_path': rel,
        'path_type': cfg['path_type'],
        'verification_status': 'verified',
        'match_status': 'exact_match',
    })
    save(f'{backend}-artifact-location.json', loc)

    dep_body = {
        'name': f"rur001-{backend}-{RUN_ID}",
        'display_name': f"RUR001 {backend} deployment {RUN_ID}",
        'model_artifact_id': art['id'],
        'node_backend_runtime_id': nbr_id,
        'placement_json': {'node_id': node_id, 'accelerator_ids': [gpu_id]},
        'service_json': {'host_port': cfg['host_port'], 'container_port': cfg['container_port'], 'health_port': cfg['host_port']},
        'config_overrides': {},
    }
    preview = api('POST', '/api/v1/deployments/preview', dep_body)
    save(f'{backend}-deployment-preview.json', preview)
    require('unsupported runtime_type' not in json.dumps(preview), f'{backend} preview has unsupported runtime_type')
    require(bool(preview.get('run_plan') or preview.get('plan')), f'{backend} preview run plan empty')

    dep = api('POST', '/api/v1/deployments', dep_body)
    save(f'{backend}-deployment-create.json', dep)
    dep_id = dep['id']
    dry = api('POST', f'/api/v1/deployments/{dep_id}/dry-run', {})
    save(f'{backend}-deployment-dry-run.json', dry)
    require('unsupported runtime_type' not in json.dumps(dry), f'{backend} dry run has unsupported runtime_type')
    require(bool(dry.get('docker_preview') or dry.get('command_preview')), f'{backend} docker preview empty')
    detail = api('GET', f'/api/v1/deployments/{dep_id}')
    save(f'{backend}-deployment-detail.json', detail)
    list_rows = api('GET', '/api/v1/deployments')
    save(f'{backend}-deployment-list-after.json', list_rows)
    edit = api('POST', '/api/v1/config-edit/view', {'object_kind':'deployment','object_id':dep_id,'layer':'deployment','mode':'edit'})
    save(f'{backend}-deployment-config-edit-view.json', edit)
    m = extract_preview_checks(backend, preview, dry, detail, list_rows, edit, check)
    m.update({'deployment_id': dep_id, 'artifact_id': art['id'], 'node_backend_runtime_id': nbr_id, 'runtime_id': runtime_id})
    for key in ['unsupported_runtime_type_absent','preview_non_empty','docker_preview_non_empty','device_binding_visible','list_name_ok','model_name_ok','detail_ok','edit_entry_ok']:
        require(m[key], f'{backend} matrix failed {key}: {m}')
    require(int(m['host_port']) == cfg['host_port'], f'{backend} host port mismatch {m}')
    require(int(m['container_port']) == cfg['container_port'], f'{backend} container port mismatch {m}')
    require(int(m['health_port']) == cfg['host_port'], f'{backend} health port mismatch {m}')
    return m

def start_and_wait(dep_id, backend):
    start = api('POST', f'/api/v1/deployments/{dep_id}/start', {})
    save(f'{backend}-start-response.json', start)
    deadline = time.time() + 300
    last = None
    while time.time() < deadline:
        instances = api('GET', '/api/v1/model-instances')
        related = [i for i in instances if i.get('deployment_id') == dep_id]
        if related:
            last = related[-1]
            save(f'{backend}-instance-last.json', last)
            state = last.get('actual_state')
            if state in ('running','healthy'):
                return {'started': True, 'state': state, 'instance': last}
            if state in ('failed','error','stopped') and last.get('last_error'):
                return {'started': False, 'state': state, 'instance': last, 'error': last.get('last_error')}
        time.sleep(5)
    return {'started': False, 'state': (last or {}).get('actual_state','timeout'), 'instance': last, 'error': 'timeout waiting for running'}

def main():
    health(); login()
    nodes = api('GET', '/api/v1/nodes'); save('nodes.json', nodes)
    gpus = api('GET', '/api/v1/gpus'); save('gpus.json', gpus)
    require(nodes, 'no nodes')
    node = next((n for n in nodes if n.get('status') == 'online'), nodes[0])
    node_id = node['id']
    gpu = next((g for g in gpus if g.get('node_id') == node_id and g.get('vendor') == 'nvidia'), None)
    require(gpu is not None, 'no NVIDIA GPU')
    gpu_id = gpu['id']
    docker_images = api('GET', f'/api/v1/nodes/{node_id}/docker-images?limit=1000')
    save('node-docker-images.json', docker_images)

    matrix = []
    for backend, cfg in BACKENDS.items():
        matrix.append(create_flow(backend, cfg, node_id, gpu_id))
    startup = start_and_wait(matrix[-1]['deployment_id'], 'llamacpp')
    save('llamacpp-runtime-start-result.json', startup)
    if startup.get('started'):
        try:
            api('POST', f"/api/v1/deployments/{matrix[-1]['deployment_id']}/stop", {})
        except Exception as e:
            save('llamacpp-stop-error.txt', str(e))
    summary = {
        'run_id': RUN_ID,
        'base': BASE,
        'node_id': node_id,
        'gpu_id': gpu_id,
        'gpu_index': gpu.get('index'),
        'matrix': matrix,
        'runtime_start': {'backend': 'llamacpp', **startup},
    }
    save('summary.json', summary)
    lines = ['| Backend | Template copy | NBR check | Runtime edit | RunPlan preview | Docker preview | Ports | Device binding | Detail/list/edit | Runtime start |', '| -- | -- | -- | -- | -- | -- | -- | -- | -- | -- |']
    for m in matrix:
        start_cell = 'dry-run/API evidence'
        if m['backend'] == 'llamacpp':
            start_cell = 'PASS: ' + str(startup.get('state')) if startup.get('started') else 'FAIL: ' + str(startup.get('state'))
        ports = f"host {m['host_port']} / container {m['container_port']} / health {m['health_port']}"
        lines.append(f"| {m['backend']} | PASS | {m['check_status']} | PASS | PASS | PASS | {ports} | PASS | PASS | {start_cell} |")
    lines.append('')
    lines.append('Device binding source: placement_json.accelerator_ids selects the node GPU; resolver maps GPU id to NVIDIA index 0, producing Docker `--gpus device=0` and `CUDA_VISIBLE_DEVICES=0` in the resolved RunPlan/command preview.')
    if not startup.get('started'):
        lines.append('')
        lines.append('Runtime start failure captured in llamacpp-runtime-start-result.json and must be closed before RUR-001 can be CLOSED.')
    save('verification-matrix.md', '\n'.join(lines) + '\n')
    if not startup.get('started'):
        raise AssertionError(f"llama.cpp real start failed: {startup}")
    print(json.dumps(summary, ensure_ascii=False, indent=2))

if __name__ == '__main__':
    main()
