#!/usr/bin/env python3
"""Isolated Release Workspace API/CLI tests. Does not modify product code."""

from __future__ import annotations

import hashlib
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

BASE = os.environ.get("DOCKERUN_URL", "http://127.0.0.1:18080").rstrip("/")
DATA_DIR = Path(os.environ.get("DATA_DIR", "/tmp/dockerun-isolated-2222"))
EVIDENCE = Path(os.environ.get("EVIDENCE_DIR", "/workspace/artifacts/evidence"))
CLI = os.environ.get("DOCKERUN_CLI", "/workspace/bin/dockerun")
ADMIN_PASSWORD = os.environ.get("DOCKERUN_ADMIN_PASSWORD", "TestAdmin2222!")
BUNDLE_DIR = Path(os.environ.get("BUNDLE_DIR", "/tmp/dockerun-bundles-2222"))
TIMEOUT = int(os.environ.get("HTTP_TIMEOUT", "120"))

EVIDENCE.mkdir(parents=True, exist_ok=True)
BUNDLE_DIR.mkdir(parents=True, exist_ok=True)

results: list[dict] = []
token = ""
state: dict = {}


def encrypt_password(password: str) -> str:
    key = ("dockerun-aes-key-2025-secret-key".ljust(32, "0")[:32]).encode()
    iv = ("dockerun-aes-iv16".ljust(16, "0")[:16]).encode()
    out = subprocess.check_output(
        ["openssl", "enc", "-aes-256-cbc", "-K", key.hex(), "-iv", iv.hex(), "-base64", "-A"],
        input=password.encode(),
    )
    return out.decode().strip()


def record(case: str, status: str, **extra):
    row = {"case": case, "status": status, "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), **extra}
    results.append(row)
    print(json.dumps(row, ensure_ascii=False), flush=True)
    return row


def http(method: str, path: str, body=None, auth=True, raw=False, headers=None, expect=None):
    url = path if path.startswith("http") else BASE + path
    data = None
    req_headers = {"Accept": "application/json"}
    if body is not None and not isinstance(body, (bytes, bytearray)):
        data = json.dumps(body).encode()
        req_headers["Content-Type"] = "application/json"
    elif isinstance(body, (bytes, bytearray)):
        data = body
    if headers:
        req_headers.update(headers)
    if auth and token:
        req_headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(url, data=data, method=method, headers=req_headers)
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            raw_body = resp.read()
            parsed = raw_body if raw else (json.loads(raw_body.decode() or "null") if raw_body else None)
            result = {"code": resp.status, "body": parsed if not raw else raw_body[:4000].decode("utf-8", "replace")}
    except urllib.error.HTTPError as e:
        raw_body = e.read()
        try:
            parsed = json.loads(raw_body.decode() or "null")
        except Exception:
            parsed = raw_body.decode("utf-8", "replace")
        result = {"code": e.code, "body": parsed}
    except Exception as e:
        result = {"code": 0, "error": str(e)}
    if expect is not None and result.get("code") not in expect if isinstance(expect, (list, tuple, set)) else result.get("code") != expect:
        pass
    return result


def save(name: str, payload):
    path = EVIDENCE / name
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2, default=str), encoding="utf-8")
    return str(path)


def run(cmd, env=None, cwd=None, timeout=180):
    merged = os.environ.copy()
    if env:
        merged.update(env)
    p = subprocess.run(cmd, cwd=cwd, env=merged, capture_output=True, text=True, timeout=timeout)
    return {"cmd": cmd, "code": p.returncode, "stdout": p.stdout[-8000:], "stderr": p.stderr[-8000:]}


def wait_health(seconds=60):
    deadline = time.time() + seconds
    last = None
    while time.time() < deadline:
        last = http("GET", "/health", auth=False)
        if last.get("code") == 200:
            return last
        time.sleep(1)
    return last or {"code": 0, "error": "timeout"}


def wait_bundle(bundle_id, seconds=180):
    deadline = time.time() + seconds
    last = None
    while time.time() < deadline:
        listed = http("GET", "/api/bundles")
        last = listed
        if listed.get("code") == 200:
            for row in listed["body"] or []:
                if row.get("id") == bundle_id:
                    if row.get("status") == "ready":
                        return row
                    if row.get("status") == "failed":
                        return row
        time.sleep(1)
    return {"status": "timeout", "last": last}


def wait_deployment(dep_id, seconds=180):
    deadline = time.time() + seconds
    last = None
    while time.time() < deadline:
        last = http("GET", f"/api/deployments/{dep_id}")
        if last.get("code") == 200:
            status = (last["body"] or {}).get("deployment", {}).get("status")
            if status in {"success", "failed", "cancelled", "needs_review"}:
                return last
        time.sleep(2)
    return last or {"code": 0, "error": "timeout"}


def main():
    global token

    health = wait_health()
    save("s0_health.json", health)
    record("S0.3 health", "pass" if health.get("code") == 200 else "fail", **health)

    setup = http("GET", "/api/auth/check-setup", auth=False)
    save("s0_check_setup.json", setup)
    if setup.get("code") == 200 and (setup.get("body") or {}).get("require_setup"):
        created = http("POST", "/api/auth/setup", {"new_password": encrypt_password(ADMIN_PASSWORD)}, auth=False)
        save("s0_setup.json", created)
        record("S0.4 setup", "pass" if created.get("code") == 200 else "fail", **created)
    else:
        record("S0.4 setup", "pass", note="already initialized", **setup)

    login = http("POST", "/api/auth/login", {"username": "admin", "password": encrypt_password(ADMIN_PASSWORD)}, auth=False)
    save("s0_login.json", login)
    if login.get("code") == 200 and (login.get("body") or {}).get("token"):
        token = login["body"]["token"]
        record("S0.4 login", "pass", code=200, user=(login["body"].get("user") or {}).get("username"))
    else:
        record("S0.4 login", "fail", **login)
        save("results.json", results)
        return 1

    tok = http("POST", "/api/auth/tokens", {"name": "release-workspace-2222", "scopes": ["pull", "push", "deploy"], "days": 7})
    save("s0_token.json", {k: ("***" if k == "token" else v) for k, v in (tok.get("body") or {}).items()} if tok.get("code") == 201 else tok)
    if tok.get("code") == 201:
        state["cli_token"] = tok["body"]["token"]
        record("S0.4 automation token", "pass", code=201, scopes=tok["body"].get("credential", {}).get("scopes"))
    else:
        record("S0.4 automation token", "fail", **tok)

    overview0 = http("GET", "/api/overview")
    save("s1_overview_before.json", overview0)
    record("S1.1 overview empty-or-baseline", "pass" if overview0.get("code") == 200 and "counts" in (overview0.get("body") or {}) else "fail", **overview0)

    blank = http("POST", "/api/applications", {"name": "blank-app-2222", "description": "空白应用隔离测试"})
    save("s2_blank_app.json", blank)
    record("S2.1 create blank application", "pass" if blank.get("code") == 201 else "fail", **blank)
    if blank.get("code") == 201:
        state["blank_id"] = blank["body"]["id"]

    sources = http("GET", f"/api/applications/{state.get('blank_id', 1)}/sources")
    save("s2_sources.json", sources)
    builtin_reg = 0
    if sources.get("code") == 200:
        for reg in sources["body"].get("registries") or []:
            if reg.get("kind") == "builtin":
                builtin_reg = reg["id"]
                state["builtin_registry"] = reg
        record("S2 sources", "pass", builtin_registry=builtin_reg, registries=len(sources["body"].get("registries") or []))
    else:
        record("S2 sources", "fail", **sources)

    if state.get("blank_id"):
        env = http("PUT", f"/api/applications/{state['blank_id']}/environments", {
            "name": "test",
            "target_id": 1,
            "registry_id": builtin_reg,
            "parameters": {},
            "stateless": True,
            "auto_deploy": False,
            "script_mode": "inherit",
            "target_ids": "[]",
            "allowed_publishers": "[]",
            "tag_pattern": "*",
        })
        save("s2_blank_env.json", env)
        record("S2.2 blank environment", "pass" if env.get("code") == 200 else "fail", **env)
        if env.get("code") == 200:
            state["blank_env_id"] = env["body"]["id"]

    templates = http("GET", "/api/managed-templates")
    save("s3_templates.json", templates)
    ids = [t.get("id") for t in (templates.get("body") or [])] if templates.get("code") == 200 else []
    record("S3.1 list managed templates", "pass" if templates.get("code") == 200 and {"mysql", "redis", "web-app"} <= set(ids) else "fail", ids=ids, code=templates.get("code"))
    if "nginx-proxy" not in ids:
        record("S3.1 nginx-proxy managed template", "skip", note="nginx-proxy 仅在旧 templates/ 脚本目录，不在 GET /api/managed-templates")

    for tid, name, params in [
        ("redis", "redis-from-template-2222", {"REDIS_IMAGE": "redis:8.2.0", "REDIS_PORT": "16379", "REDIS_PASSWORD": "RedisPass2222"}),
        ("mysql", "mysql-from-template-2222", {"MYSQL_IMAGE": "mysql:8.0.44", "MYSQL_PORT": "13306", "MYSQL_ROOT_PASSWORD": "MysqlPass2222"}),
        ("web-app", "webapp-from-template-2222", {"IMAGE": "dockerun-test/demo:v1", "HOST_PORT": "18081", "CONTAINER_PORT": "80"}),
    ]:
        created = http("POST", f"/api/managed-templates/{tid}/applications", {
            "name": name,
            "description": f"从模板 {tid} 创建",
            "environment": "test",
            "target_id": 1,
            "registry_id": builtin_reg,
            "parameters": params,
        })
        save(f"s2_template_{tid}.json", created)
        ok = created.get("code") == 201 and not created.get("body", {}).get("deployment")
        record(f"S2.3 create from template {tid}", "pass" if ok else "fail", **created)
        if created.get("code") == 201:
            state[f"{tid}_app_id"] = created["body"]["application"]["id"]
            state[f"{tid}_env_id"] = created["body"]["environment"]["id"]

    docker_status = http("GET", "/api/system/docker-status")
    save("s0_docker_status.json", docker_status)
    record("S0 docker-status", "pass" if docker_status.get("code") == 200 else "fail", **docker_status)

    # Push nginx into builtin registry for deploy + auto-deploy.
    if state.get("cli_token"):
        login_reg = run(["docker", "login", "-u", "admin", "--password-stdin", "127.0.0.1:18080"], env={})
        # password via stdin
        p = subprocess.run(
            ["docker", "login", "-u", "admin", "--password-stdin", "127.0.0.1:18080"],
            input=state["cli_token"],
            capture_output=True,
            text=True,
            timeout=60,
        )
        login_reg = {"code": p.returncode, "stdout": p.stdout, "stderr": p.stderr}
        save("s4_docker_login.json", login_reg)
        record("S4.5 docker login builtin", "pass" if p.returncode == 0 else "fail", **login_reg)
        tag = run(["docker", "tag", "nginx:1.28-alpine", "127.0.0.1:18080/dockerun-test/demo:v1"])
        push = run(["docker", "push", "127.0.0.1:18080/dockerun-test/demo:v1"], timeout=180)
        save("s4_push_v1.json", {"tag": tag, "push": push})
        record("S4.5 push builtin demo:v1", "pass" if push["code"] == 0 else "fail", **push)

    compose = (
        "services:\n"
        "  demo:\n"
        "    image: 127.0.0.1:18080/dockerun-test/demo:v1\n"
        "    restart: unless-stopped\n"
        "    command: [sh, -c, 'while true; do sleep 30; done']\n"
        "    healthcheck:\n"
        "      test: [CMD, 'true']\n"
        "      interval: 1s\n"
        "      timeout: 1s\n"
        "      retries: 5\n"
    )
    if state.get("blank_id") and state.get("blank_env_id"):
        saved = http("PUT", f"/api/applications/{state['blank_id']}/deployment-config", {
            "environment_id": state["blank_env_id"],
            "compose": compose,
            "configs": {},
            "execution": {},
            "scripts": {},
            "source": "registry",
            "registry_id": builtin_reg,
            "platform": "linux/amd64",
        })
        save("s4_save_config.json", saved)
        record("S4.1 save deployment config", "pass" if saved.get("code") == 200 else "fail", **saved)

        release = http("POST", f"/api/applications/{state['blank_id']}/releases?environment_id={state['blank_env_id']}", {
            "version": "v1-local-test",
            "source": "registry",
            "registry_id": builtin_reg,
            "compose": compose,
            "configs": "{}",
            "platform": "linux/amd64",
            "save_definition": True,
        })
        save("s4_create_release.json", release)
        record("S4.2 create registry release", "pass" if release.get("code") == 201 else "fail", **release)
        if release.get("code") == 201:
            state["blank_release_id"] = release["body"]["id"]

        if state.get("blank_release_id"):
            pre = http("POST", "/api/deployments/preflight", {
                "environment_id": state["blank_env_id"],
                "release_id": state["blank_release_id"],
            })
            save("s4_preflight.json", pre)
            record("S4.3 preflight", "pass" if pre.get("code") == 200 else "fail", **pre)
            dep = http("POST", "/api/deployments", {
                "environment_id": state["blank_env_id"],
                "release_id": state["blank_release_id"],
                "idempotency_key": f"manual-{int(time.time()*1000)}",
            })
            save("s4_enqueue.json", dep)
            record("S4.4 enqueue deploy", "pass" if dep.get("code") == 202 else "fail", **dep)
            if dep.get("code") == 202:
                waited = wait_deployment(dep["body"]["id"])
                save("s4_deploy_wait.json", waited)
                status = ((waited.get("body") or {}).get("deployment") or {}).get("status")
                record("S4.4 deploy result", "pass" if status == "success" else "fail", status=status, code=waited.get("code"), result=((waited.get("body") or {}).get("deployment") or {}).get("result"))
                state["blank_deployment_id"] = dep["body"]["id"]

        auto = http("PATCH", f"/api/applications/{state['blank_id']}/environments/{state['blank_env_id']}/automation", {
            "auto_deploy": True,
            "registry_id": builtin_reg,
            "repository": "dockerun-test/demo",
            "tag_pattern": "v*",
            "allowed_publishers": "[]",
        })
        save("s4_automation.json", auto)
        record("S4.5 enable auto-deploy", "pass" if auto.get("code") == 200 else "fail", **auto)

        tag2 = run(["docker", "tag", "nginx:1.28-alpine", "127.0.0.1:18080/dockerun-test/demo:v2"])
        push2 = run(["docker", "push", "127.0.0.1:18080/dockerun-test/demo:v2"], timeout=180)
        save("s4_push_v2.json", {"tag": tag2, "push": push2})
        record("S4.5 push builtin demo:v2", "pass" if push2["code"] == 0 else "fail", **push2)
        auto_seen = False
        auto_dep = None
        deadline = time.time() + 90
        while time.time() < deadline:
            listed = http("GET", "/api/deployments")
            if listed.get("code") == 200:
                for row in listed["body"] or []:
                    if row.get("automatic") and row.get("environment_id") == state.get("blank_env_id"):
                        auto_seen = True
                        auto_dep = row
                        break
            if auto_seen:
                break
            time.sleep(2)
        save("s4_auto_deployments.json", auto_dep or {"note": "no automatic deployment observed"})
        if auto_seen:
            waited = wait_deployment(auto_dep["id"])
            save("s4_auto_wait.json", waited)
            status = ((waited.get("body") or {}).get("deployment") or {}).get("status")
            record("S4.5 auto-deploy from builtin push", "pass" if status == "success" else "fail", status=status, deployment_id=auto_dep["id"])
        else:
            events = http("GET", "/api/overview")
            catalog = http("GET", "/api/builtin/catalog")
            save("s4_auto_missing.json", {"overview": events, "catalog": catalog})
            record("S4.5 auto-deploy from builtin push", "fail", note="90s 内未见 automatic 部署任务")

    record("S4.6 external registry", "skip", note="无外部仓库凭据，按计划跳过")

    listed = http("GET", "/api/deployments")
    save("s5_deployments.json", listed)
    record("S5.1 list deployments", "pass" if listed.get("code") == 200 else "fail", count=len(listed.get("body") or []) if listed.get("code") == 200 else 0, code=listed.get("code"))
    if state.get("blank_deployment_id"):
        detail = http("GET", f"/api/deployments/{state['blank_deployment_id']}")
        save("s5_deployment_detail.json", detail)
        record("S5.2 deployment detail", "pass" if detail.get("code") == 200 else "fail", **{k: detail.get(k) for k in ("code",)})

    cli_env = {
        "DOCKERUN_URL": BASE,
        "DOCKERUN_TOKEN": state.get("cli_token", ""),
    }
    if state.get("cli_token"):
        st = run([CLI, "status"], env=cli_env)
        save("s5_cli_status.txt", st)
        record("S5.3 CLI status", "pass" if st["code"] == 0 else "fail", code=st["code"], stdout=st["stdout"][:500])
        if state.get("blank_deployment_id"):
            stid = run([CLI, "status", str(state["blank_deployment_id"])], env=cli_env)
            save("s5_cli_status_id.txt", stid)
            record("S5.3 CLI status id", "pass" if stid["code"] == 0 else "fail", code=stid["code"])

    # Offline package using local nginx compose.
    compose_file = BUNDLE_DIR / "compose.yaml"
    compose_file.write_text(
        "services:\n"
        "  fixture:\n"
        "    image: nginx:1.28-alpine\n"
        "    command: [sh, -c, 'while true; do sleep 30; done']\n"
        "    healthcheck:\n"
        "      test: [CMD, 'true']\n"
        "      interval: 1s\n"
        "      timeout: 1s\n"
        "      retries: 5\n",
        encoding="utf-8",
    )
    archive = BUNDLE_DIR / "blank-app-2222-release.tar.gz"
    packed = run([CLI, "package", "blank-app-2222", "--file", str(compose_file), "--platform", "linux/amd64", "--output", str(archive), "--version", "offline-v1"], env=cli_env, timeout=180)
    save("s6_package.txt", packed)
    record("S6.1 CLI package", "pass" if packed["code"] == 0 and archive.exists() else "fail", code=packed["code"], exists=archive.exists(), size=archive.stat().st_size if archive.exists() else 0, stderr=packed["stderr"][-400:])

    if archive.exists() and state.get("cli_token"):
        imported = run([CLI, "import", str(archive), "--wait"], env=cli_env, timeout=180)
        save("s6_import.txt", imported)
        record("S6.2 CLI import", "pass" if imported["code"] == 0 else "fail", code=imported["code"], stdout=imported["stdout"][-800:], stderr=imported["stderr"][-400:])
        bundles = http("GET", "/api/bundles")
        save("s6_bundles.json", bundles)
        ready = None
        if bundles.get("code") == 200:
            for row in bundles["body"] or []:
                if row.get("status") == "ready":
                    ready = row
                    break
        if ready:
            ver = http("POST", f"/api/bundles/{ready['id']}/version", {"application_id": state.get("blank_id")})
            save("s6_bundle_version.json", ver)
            record("S6.3 bundle version", "pass" if ver.get("code") in {200, 201} else "fail", **ver)
            deployed = run([CLI, "deploy", "blank-app-2222", "--env", "test", "--bundle", str(archive), "--wait"], env=cli_env, timeout=240)
            save("s6_deploy_bundle.txt", deployed)
            record("S6.4 CLI deploy from bundle", "pass" if deployed["code"] == 0 else "fail", code=deployed["code"], stdout=deployed["stdout"][-800:], stderr=deployed["stderr"][-400:])
            pub = http("POST", "/api/bundles/publish", {"release_id": (ver.get("body") or {}).get("release", {}).get("id")})
            save("s6_publish.json", pub)
            record("S6.5 publish bundle to builtin", "pass" if pub.get("code") in {200, 201, 202} else "fail", **pub)
        else:
            record("S6.3 bundle version", "fail", note="no ready bundle after import")

    # Custom template from saved blank app definition
    if state.get("blank_id"):
        tmpl = http("POST", f"/api/applications/{state['blank_id']}/template", {
            "name": "custom-demo-2222",
            "version": "1.0.0",
            "description": "由空白应用另存",
        })
        save("s3_save_custom_template.json", tmpl)
        record("S3.3 save custom template", "pass" if tmpl.get("code") == 201 else "fail", **tmpl)
        listed_t = http("GET", "/api/managed-templates")
        custom_ids = [t.get("id") for t in (listed_t.get("body") or []) if str(t.get("id", "")).startswith("custom-")]
        save("s3_templates_after.json", listed_t)
        record("S3.4 custom template listed", "pass" if custom_ids else "fail", custom_ids=custom_ids)

    overview1 = http("GET", "/api/overview")
    save("s1_overview_after.json", overview1)
    before = (overview0.get("body") or {}).get("counts") or {}
    after = (overview1.get("body") or {}).get("counts") or {}
    grew = after.get("applications", 0) > before.get("applications", 0)
    record("S1.2 overview counts increased", "pass" if overview1.get("code") == 200 and grew else "fail", before=before, after=after)

    catalog = http("GET", "/api/builtin/catalog")
    save("s4_builtin_catalog.json", catalog)
    record("S4 builtin catalog", "pass" if catalog.get("code") == 200 else "fail", **catalog)

    save("results.json", results)
    save("state.json", {k: v for k, v in state.items() if k != "cli_token"})
    failed = [r for r in results if r["status"] == "fail"]
    print(f"SUMMARY pass={sum(1 for r in results if r['status']=='pass')} fail={len(failed)} skip={sum(1 for r in results if r['status']=='skip')}", flush=True)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
