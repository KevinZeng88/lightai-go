#!/usr/bin/env python3
"""LightAI Bootstrap export mode — generate YAML profile from running environment."""
import json, os, sys, subprocess, time
from datetime import datetime, timezone

def api_get(url, cookie_jar):
    args = ['curl', '-sS', url, '-H', f'Origin: {base_url}', '-H', 'Content-Type: application/json', '-b', cookie_jar]
    result = subprocess.run(args, capture_output=True, text=True)
    if result.returncode == 0 and result.stdout.strip():
        try:
            data = json.loads(result.stdout)
            return data if isinstance(data, list) else data.get('data', data.get('items', data.get('results', [])))
        except:
            return []
    return []

def snake_case(s):
    import re
    s = re.sub(r'[^a-zA-Z0-9_]', '_', s).strip('_')
    return re.sub(r'_+', '_', s).lower()

if __name__ == '__main__':
    if len(sys.argv) < 4:
        print("Usage: bootstrap-export.py <base_url> <cookie_jar> <csrf_token> <output_profile> [include_runtime]"); sys.exit(1)
    base_url = sys.argv[1].rstrip('/')
    cookie_jar = sys.argv[2]
    csrf = sys.argv[3] if sys.argv[3] != 'NONE' else None
    output_profile = sys.argv[4]
    include_runtime = sys.argv[5] == 'true' if len(sys.argv) > 5 else False
    out_dir = os.path.dirname(output_profile) or '.'
    os.makedirs(out_dir, exist_ok=True)
    ts = datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
    
    backends = api_get(f'{base_url}/api/v1/backends', cookie_jar)
    nodes = api_get(f'{base_url}/api/v1/nodes', cookie_jar)
    node = nodes[0] if nodes else {}
    artifacts = api_get(f'{base_url}/api/v1/model-artifacts', cookie_jar)
    bruntimes = api_get(f'{base_url}/api/v1/backend-runtimes', cookie_jar)
    nbrs = api_get(f'{base_url}/api/v1/nodes/{node.get("id","")}/backend-runtimes', cookie_jar) if node else []
    deployments = api_get(f'{base_url}/api/v1/deployments', cookie_jar) if include_runtime else []
    
    # Build profile
    models = {}
    for art in artifacts:
        name = art.get('name',''); path = art.get('path','')
        if not path: continue
        key = snake_case(name) or f'model_{len(models)}'
        fmt = art.get('format','custom')
        kind = 'gguf' if fmt == 'gguf' else 'huggingface'
        models[key] = {'display_name': art.get('display_name',name), 'kind': kind, 'path': path, 'artifact_id': art.get('id',''), 'source': 'export'}
    
    br_map = {br['id']: br for br in bruntimes}
    runtimes = {}
    for nbr in nbrs:
        brid = nbr.get('backend_runtime_id',''); br = br_map.get(brid, {})
        bid = br.get('backend_id',''); backend_name = bid.split('.')[-1] if '.' in bid else bid
        config = nbr.get('config_set',{})
        if isinstance(config, str):
            try: config = json.loads(config)
            except: config = {}
        model_path = config.get('model_path','')
        model_key = ''
        for mk, mv in models.items():
            if mv.get('path','') == model_path: model_key = mk; break
        if not model_key and models: model_key = list(models.keys())[0]
        key = backend_name if backend_name else f'rt_{brid[:8]}'
        if key in runtimes: key = f'{key}_2'
        runtimes[key] = {
            'backend': backend_name, 'image': nbr.get('image_ref','') or br.get('image_ref',''),
            'model': model_key, 'container_port': 8000,
            'host_port': nbr.get('host_port',8000), 'backend_runtime_id': brid,
            'node_backend_runtime_id': nbr.get('id',''), 'parameters': {},
            'config_set': config, 'config_overrides': {},
            'health_check': {}, 'status': nbr.get('status','unknown')
        }
    
    # Write profile
    def w(f, k, v, i=0):
        p='  '*i
        if isinstance(v, bool): f.write(f'{p}{k}: {"true" if v else "false"}\n')
        elif v is None or v == '': f.write(f'{p}{k}: ""\n')
        elif isinstance(v, str): f.write(f'{p}{k}: {v}\n')
        elif isinstance(v, (int,float)): f.write(f'{p}{k}: {v}\n')
        elif isinstance(v, dict):
            if not v: f.write(f'{p}{k}: {{}}\n'); return
            f.write(f'{p}{k}:\n')
            for sk, sv in v.items(): w(f, sk, sv, i+1)
        elif isinstance(v, list):
            if not v: f.write(f'{p}{k}: []\n'); return
            f.write(f'{p}{k}:\n')
            for item in v:
                if isinstance(item, dict):
                    for dk, dv in item.items(): w(f, dk, dv, i+1)
                elif isinstance(item, str): f.write(f'{p}  - {item}\n')
                else: f.write(f'{p}  - {item}\n')
        else: f.write(f'{p}{k}: {v}\n')
    
    with open(output_profile, 'w') as f:
        f.write(f'# LightAI Bootstrap Profile — exported from {base_url}\n# Generated: {ts}\n\n')
        f.write(f'profile_name: exported-local\n\n')
        for s in ['server','auth','tenant','node']:
            f.write(f'{s}:\n')
            d = {
                'server': {'base_url': base_url, 'agent_url': 'http://localhost:19091', 'runtime_dir': '/tmp/lightai'},
                'auth': {'username': 'admin', 'initial_password_env': 'LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD', 'initial_password': '', 'initial_password_file': '', 'initial_password_runtime_files': ['auto'], 'final_password_env': 'LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD', 'final_password_file': ''},
                'tenant': {'name': 'default'},
                'node': {'id': node.get('id',''), 'name': node.get('hostname',''), 'gpu_vendor': 'nvidia', 'gpu_ids': ['0'], 'accelerator_ids': ['0']} if node else {},
            }[s]
            for k, v in d.items(): w(f, k, v, 1)
            f.write('\n')
        f.write('models:\n')
        for mk, mv in sorted(models.items()):
            f.write(f'  {mk}:\n')
            for k in ['display_name','kind','path','artifact_id','source']:
                if k in mv: f.write(f'    {k}: {mv[k]}\n')
        f.write('\nruntimes:\n')
        for rk, rv in sorted(runtimes.items()):
            f.write(f'  {rk}:\n')
            for k in ['backend','image','model','container_port','host_port','parameters','status']:
                if k in rv:
                    if isinstance(rv[k], dict) and not rv[k]: f.write(f'    {k}: {{}}\n')
                    elif isinstance(rv[k], dict):
                        f.write(f'    {k}:\n')
                        for pk, pv in rv[k].items():
                            if isinstance(pv, dict) and pv:
                                f.write(f'      {pk}:\n')
                                for spk, spv in pv.items(): f.write(f'        {spk}: {spv}\n')
                            else: f.write(f'      {pk}: {pv}\n')
                    else: f.write(f'    {k}: {rv[k]}\n')
        f.write('\nbootstrap:\n')
        for k, v in {'default_mode': 'dry-run', 'output_dir': '/tmp/lightai/e2e/bootstrap', 'allow_real_container_start': False, 'allow_chat_completion': False, 'keep_containers_after_full': False, 'default_export_profile': output_profile, 'include_runtime_state': bool(include_runtime)}.items():
            w(f, k, v, 1)
        f.write('\n')
        if include_runtime:
            f.write('# Runtime state\n')
            for d in deployments:
                f.write(f'# deployment: {d.get("name","")} id={d.get("id","")[:20]}\n')
    
    # Summary files
    with open(f'{out_dir}/export-summary.json', 'w') as f:
        json.dump({'profile_path': os.path.abspath(output_profile), 'backup_path': '', 'tenants': 1, 'nodes': len(nodes), 'models': len(models), 'model_locations': len(models), 'runtimes': len(runtimes), 'node_backend_runtimes': len(nbrs), 'deployments': len(deployments), 'warnings': 0, 'generated_at': ts}, f, indent=2)
    with open(f'{out_dir}/export-resource-map.json', 'w') as f:
        rm = {mk: {'artifact_id': mv.get('artifact_id','')} for mk, mv in models.items()}
        for rk, rv in runtimes.items(): rm[rk] = {'backend_runtime_id': rv.get('backend_runtime_id',''), 'node_backend_runtime_id': rv.get('node_backend_runtime_id','')}
        json.dump(rm, f, indent=2)
    with open(f'{out_dir}/export-warnings.json', 'w') as f:
        wl = []
        if not models: wl.append({'type': 'no_models', 'message': 'No model artifacts found'})
        if len(runtimes) < 3: wl.append({'type': 'incomplete_runtimes', 'message': f'Only {len(runtimes)} runtimes found (expected 3)'})
        json.dump(wl, f, indent=2)
    
    print(f'exported {len(models)} models, {len(runtimes)} runtimes, {len(nbrs)} NBRs to {output_profile}')
