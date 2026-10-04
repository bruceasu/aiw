"""FD-020 public HTTP/storage contract checks; requires recorded Planner approval."""
import datetime
import http.client
import hashlib
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import threading
import time
import traceback

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / 'docs/features/reports/FD-020-test-raw-r2.json'
RESULTS = []

def case(name, fn):
    try:
        fn()
        RESULTS.append({'id': name, 'passed': True})
    except Exception as exc:
        RESULTS.append({'id': name, 'passed': False, 'error': type(exc).__name__ + ': ' + str(exc), 'traceback': traceback.format_exc()})

def require(condition, message):
    if not condition:
        raise AssertionError(message)

with tempfile.TemporaryDirectory(prefix='aiw-fd020-') as task_dir:
    tmp = Path(task_dir)
    exe = tmp / 'gateway.exe'
    env = {k: v for k, v in os.environ.items() if k.upper() not in
           {'OPENAI_API_KEY', 'OPENAI_BASE_URL', 'CODEX_HOME', 'AIW_AI_API_KEY'}}
    env.update(GOPROXY='off', GOSUMDB='off', GOTOOLCHAIN='local', CGO_ENABLED='0')
    built = subprocess.run(['go', 'build', '-o', str(exe), '.'],
                           cwd=ROOT / 'program/agent-gateway', env=env,
                           capture_output=True, text=True, timeout=120)
    if built.returncode:
        EVIDENCE.write_text(json.dumps({'build_exit': built.returncode,
            'build_error': built.stderr, 'cases': []}, ensure_ascii=False, indent=2), encoding='utf-8')
        raise SystemExit(built.returncode)
    node = shutil.which('node')
    require(node is not None, 'Node executable required for temporary backend')
    fake = tmp / 'fake-codex.js'
    fake.write_text('setTimeout(() => process.exit(1), 250);', encoding='utf-8')
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        port = sock.getsockname()[1]
    keys = ['A' * 64, 'B' * 64, 'C' * 64, 'D' * 64]
    config = {'mode': 'development', 'listen': f'127.0.0.1:{port}',
        'state_dir': str(tmp / 'state'), 'workspace_dir': str(tmp / 'work'),
        'codex_path': node, 'codex_script': str(fake), 'timezone': 'UTC',
        'timeout_seconds': 1, 'global_concurrency': 2,
        'models': {'gateway-default': 'test-model'},
        'principals': [{'id': 'caller-a', 'enabled': True, 'keys': [keys[0]],
            'allowed_models': ['gateway-default'], 'rpm': 100, 'daily_limit': 100, 'concurrency': 1},
            {'id': 'caller-b', 'enabled': True, 'keys': [keys[1]],
            'allowed_models': ['gateway-default'], 'rpm': 100, 'daily_limit': 100, 'concurrency': 1},
            {'id': 'disabled', 'enabled': False, 'keys': [keys[2]],
            'allowed_models': ['gateway-default'], 'rpm': 100, 'daily_limit': 100, 'concurrency': 1}]}
    cfg = tmp / 'gateway.json'
    proc = None
    log_handles = []
    logs = []

    def stop():
        global proc
        if proc is not None:
            proc.terminate()
            proc.wait(timeout=10)
            proc = None
        for handle in log_handles:
            handle.close()
        log_handles.clear()
        lock = tmp / 'state/gateway.lock'
        if lock.exists():
            lock.unlink()  # Test-owned state; backend cases have already returned.

    def start(expect_failure=False):
        global proc
        cfg.write_text(json.dumps(config), encoding='utf-8')
        log = tmp / f'log-{len(logs)}.txt'
        logs.append(log)
        handle = log.open('w', encoding='utf-8')
        log_handles.append(handle)
        proc = subprocess.Popen([str(exe), 'start', '--config', str(cfg)],
                                cwd=tmp, env=env, stdout=handle, stderr=handle)
        until = time.monotonic() + 5
        while time.monotonic() < until:
            if proc.poll() is not None:
                return False
            if not expect_failure:
                try:
                    with socket.create_connection(('127.0.0.1', port), timeout=.1):
                        return True
                except OSError:
                    pass
            time.sleep(.05)
        return False

    def request(path='/v1/models', key=keys[0], method='GET', payload=None):
        conn = http.client.HTTPConnection('127.0.0.1', port, timeout=8)
        headers = {'Authorization': 'Bearer ' + key} if key else {}
        if payload is not None:
            headers['Content-Type'] = 'application/json'
        conn.request(method, path, json.dumps(payload) if payload is not None else None, headers)
        response = conn.getresponse()
        body = response.read().decode('utf-8')
        rid = response.getheader('x-request-id')
        status = response.status
        conn.close()
        for _ in range(50):
            record_path = tmp / 'state/requests' / (str(rid) + '.json')
            if record_path.exists():
                record = json.loads(record_path.read_text(encoding='utf-8'))
                if record.get('state') != 'in_progress':
                    break
            time.sleep(.01)
        else:
            raise AssertionError('terminal request record unavailable')
        return status, json.loads(body), rid, record

    def check_request(name, expected_status, principal, **kwargs):
        status, body, rid, record = request(**kwargs)
        require(status == expected_status if isinstance(expected_status, int) else status in expected_status, f'unexpected HTTP status: {status}')
        require(bool(rid) and record['request_id'] == rid, 'request id mismatch')
        require(record.get('principal', '') == principal, 'principal mismatch')
        require(record.get('http_status') == status, 'persistent status mismatch')
        require(record.get('duration_ms') is not None, 'known duration missing')
        return record

    try:
        require(start(), 'gateway failed to start')
        records = {}
        case('models_success', lambda: records.update(models=check_request('models', 200, 'caller-a')))
        case('unauthenticated', lambda: records.update(unauth=check_request('unauth', 401, '', key='')))
        case('disabled_key', lambda: check_request('disabled', 401, '', key=keys[2]))
        case('unknown_key', lambda: check_request('unknown', 401, '', key='E' * 64))
        case('former_digest_key', lambda: check_request('opaque', 200, 'caller-a', key=keys[0]))
        case('digest_conversion_not_accepted', lambda: check_request('digest', 401, '', key=hashlib.sha256(keys[0].encode()).hexdigest()))
        case('parameter_rejection', lambda: records.update(rejected=check_request('invalid', 400, 'caller-a',
            path='/v1/responses', method='POST', payload={'model': 'gateway-default', 'input': 'SECRET_PROMPT', 'cwd': 'SECRET_PATH'})))
        case('model_rejection', lambda: check_request('model', {400, 403, 404}, 'caller-a', path='/v1/responses',
            method='POST', payload={'model': 'unconfigured-SECRET_MODEL', 'input': 'SECRET_PROMPT'}))
        case('usage_query_rejection', lambda: check_request('query', 400, 'caller-a', path='/v1/usage?start_date=invalid-SECRET_QUERY'))
        case('unknown_route', lambda: require(check_request('route', 404, 'caller-a', path='/SECRET_PATH?x=SECRET_QUERY')['route'] == 'other', 'unknown route not sanitized'))
        case('unknown_method', lambda: check_request('method', 405, 'caller-a', method='POST'))
        case('unknown_method_normalization', lambda: require(check_request('method', {404, 405}, 'caller-a', method='BREW')['method'] == 'other', 'unknown method not normalized'))
        case('caller_b_models', lambda: check_request('b', 200, 'caller-b', key=keys[1]))
        def usage_isolation():
            status, body, _, _ = request('/v1/usage')
            require(status == 200 and body['principal'] == 'caller-a', 'usage principal mismatch')
            require(body['http_groups'] and not body['groups'], 'pre-execution groups incorrect')
            require(sum(g['executed'] for g in body['http_groups']) == 0, 'rejection counted execution')
            require(sum(g['requests'] for g in body['http_groups'] if g['route'] == records['models']['route'] and g['method'] == 'GET') == 2, 'other callers leaked into models group')
            require(any(g['method'] == 'POST' and g['rejected'] for g in body['http_groups']), 'POST rejection group missing')
            require(any(g['method'] == 'other' and g['rejected'] for g in body['http_groups']), 'unknown method rejection group missing')
            require(any(g['in_progress'] for g in body['http_groups']), 'usage query not tracked in progress')
            require(body['today_used'] == 0, 'rejection consumed quota')
        case('usage_isolation_and_no_execution', usage_isolation)
        def http_aggregation():
            _, body, _, _ = request('/v1/usage')
            groups = body['http_groups']
            require(sum(g['rejected'] for g in groups) >= 3, 'HTTP rejections not grouped')
            require(sum(g['duration_samples'] for g in groups) == sum(g['requests'] - g['in_progress'] for g in groups), 'known terminal duration sample count incorrect')
            require(all(g['total_duration_ms'] >= 0 for g in groups), 'negative duration totals')
            require(any(g['error_codes'] for g in groups), 'rejection error codes missing')
            require(any('400' in g['http_statuses'] for g in groups), 'HTTP status count missing')
        case('http_status_error_duration_aggregation', http_aggregation)
        def backend_failure():
            r = check_request('backend', 502, 'caller-a', path='/v1/responses', method='POST',
                payload={'model': 'gateway-default', 'input': 'SECRET_PROMPT'})
            require(r['state'] == 'failed' and r['execution_started'], 'failure execution state missing')
            require((tmp / 'state/metadata' / (r['request_id'] + '.json')).exists(), 'execution id linkage missing')
        case('backend_failure_and_linkage', backend_failure)
        fake.write_text('setTimeout(() => process.exit(1), 5000);', encoding='utf-8')
        def backend_timeout():
            r = check_request('timeout', 504, 'caller-a', path='/v1/responses', method='POST',
                payload={'model': 'gateway-default', 'input': 'SECRET_PROMPT'})
            require(r['state'] == 'timed_out' and r['execution_started'], 'timeout state missing')
        case('backend_timeout', backend_timeout)
        def concurrency_rejection():
            running = []
            thread = threading.Thread(target=lambda: running.append(request('/v1/responses', method='POST',
                payload={'model': 'gateway-default', 'input': 'SECRET_PROMPT'})))
            thread.start()
            time.sleep(.2)
            try:
                r = check_request('concurrency', {429, 503}, 'caller-a', path='/v1/responses', method='POST',
                    payload={'model': 'gateway-default', 'input': 'SECRET_PROMPT'})
                require(r['state'] == 'rejected' and not r['execution_started'] and bool(r['error_code']), 'concurrency rejection state/error incorrect')
            finally:
                thread.join(timeout=8)
            require(bool(running), 'parallel request did not finish')
        case('concurrency_rejection', concurrency_rejection)
        fake.write_text('setTimeout(() => process.exit(1), 250);', encoding='utf-8')
        def unknown_usage():
            _, body, _, _ = request('/v1/usage')
            require(body['today_used'] == 3, 'started failure/timeout quota incorrect')
            require(sum(g['unknown_usage_requests'] for g in body['groups']) == 3, 'unknown token accounting incorrect')
        case('failed_execution_quota_unknown_tokens', unknown_usage)
        def log_privacy():
            text = ''.join(p.read_text(encoding='utf-8') for p in logs)
            request_text = ''.join(p.read_text(encoding='utf-8') for p in (tmp / 'state/requests').glob('*.json'))
            for secret in keys + ['SECRET_PROMPT', 'SECRET_QUERY', 'SECRET_PATH', 'SECRET_MODEL', 'Authorization']:
                require(secret not in text + request_text, 'sensitive field disclosed')
            rid = records['models']['request_id']
            events = [json.loads(line[line.index('{'):]) for line in text.splitlines() if '{' in line]
            matching = [e for e in events if e.get('request_id') == rid]
            require({e.get('event') for e in matching} >= {'request_started', 'request_finished'}, 'start/finish log linkage missing')
        case('log_linkage_and_privacy', log_privacy)
        stop()
        config['principals'][0]['keys'] = [keys[3]]
        require(start(), 'rotation restart failed')
        case('rotation_old_key_rejected', lambda: check_request('old', 401, '', key=keys[0]))
        case('rotation_new_key', lambda: check_request('new', 200, 'caller-a', key=keys[3]))
        case('rotation_quota_preserved', lambda: require(request('/v1/usage', key=keys[3])[1]['today_used'] == 3, 'rotation cleared quota'))
        stop()
        config['principals'][0]['rpm'] = 1
        require(start(), 'RPM restart failed')
        request('/v1/responses', key=keys[3], method='POST', payload={'model': 'gateway-default', 'input': 'SECRET_PROMPT'})
        case('rpm_rejection', lambda: require(not check_request('rpm', 429, 'caller-a', path='/v1/responses',
            key=keys[3], method='POST', payload={'model': 'gateway-default', 'input': 'SECRET_PROMPT'})['execution_started'], 'RPM rejection started execution'))
        stop()
        rec_path = tmp / 'state/requests' / (records['models']['request_id'] + '.json')
        rec = json.loads(rec_path.read_text(encoding='utf-8'))
        rec.update(state='in_progress', finished_at=None, duration_ms=None, http_status=0, error_code='')
        rec_path.write_text(json.dumps(rec), encoding='utf-8')
        require(start(), 'recovery restart failed')
        def recovery():
            r = json.loads(rec_path.read_text(encoding='utf-8'))
            require(r['state'] == 'interrupted' and r['error_code'] == 'request_interrupted', 'interruption not recovered')
            require(r['finished_at'] is None and r['duration_ms'] is None, 'invented interruption timing')
        case('interrupted_recovery', recovery)
        stop()
        config['principals'][1]['keys'] = [keys[3]]
        case('duplicate_keys_rejected', lambda: require(not start(True) and proc.poll() is not None, 'duplicate keys accepted'))
        stop()
        config['principals'][1]['keys'] = [keys[1]]
        config['principals'][0]['key_hashes'] = []
        case('key_hashes_rejected', lambda: require(not start(True) and proc.poll() is not None, 'removed field accepted'))
        stop()
        del config['principals'][0]['key_hashes']
        manifest = tmp / 'state/requests-manifest.json'
        original = manifest.read_bytes()
        manifest.write_text('{broken', encoding='utf-8')
        case('corrupt_manifest_rejected', lambda: require(not start(True) and proc.poll() is not None, 'corrupt store accepted'))
        stop()
        manifest.write_bytes(original)
        (tmp / 'state/requests').rename(tmp / 'state/requests-saved')
        case('missing_upgraded_directory_rejected', lambda: require(not start(True) and proc.poll() is not None, 'missing upgraded requests accepted'))
    finally:
        stop()
        EVIDENCE.write_text(json.dumps({'build_exit': 0, 'cases': RESULTS,
            'passed': sum(x['passed'] for x in RESULTS), 'failed': sum(not x['passed'] for x in RESULTS),
            'coverage': 'branch coverage not measured; black-box process has no branch instrumentation'},
            ensure_ascii=False, indent=2), encoding='utf-8')
print(json.dumps({'passed': sum(x['passed'] for x in RESULTS), 'failed': sum(not x['passed'] for x in RESULTS)}))
raise SystemExit(1 if any(not x['passed'] for x in RESULTS) else 0)
